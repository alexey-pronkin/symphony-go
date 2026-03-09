package insights

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DefaultInspector struct {
	Local  GitGoInspector
	Client *http.Client
}

func (i DefaultInspector) Inspect(
	ctx context.Context,
	source SourceConfig,
	staleAfter time.Duration,
	now time.Time,
) (SCMSourceMetrics, error) {
	metrics := SCMSourceMetrics{
		Kind:       source.Kind,
		Name:       source.Name,
		RepoPath:   source.RepoPath,
		MainBranch: source.MainBranch,
		Repository: source.Repository,
		ProjectID:  source.ProjectID,
		Warnings:   []string{},
	}

	hasAnyData := false
	if strings.TrimSpace(source.RepoPath) != "" {
		localMetrics, err := i.Local.Inspect(ctx, source, staleAfter, now)
		if err != nil {
			metrics.Warnings = append(metrics.Warnings, fmt.Sprintf("local git metrics unavailable: %v", err))
		} else {
			metrics = mergeSCMMetrics(metrics, localMetrics)
			hasAnyData = true
		}
	}

	providerMetrics, supported, err := i.inspectProvider(ctx, source, staleAfter, now)
	switch {
	case err != nil:
		metrics.Warnings = append(metrics.Warnings, err.Error())
	case supported:
		metrics = mergeSCMMetrics(metrics, providerMetrics)
		hasAnyData = true
	default:
		metrics.Warnings = append(metrics.Warnings, fmt.Sprintf("%s provider metrics are not available yet", source.Kind))
	}

	if !hasAnyData {
		return SCMSourceMetrics{}, errors.New(strings.Join(metrics.Warnings, "; "))
	}
	metrics.MergeReadiness = clampScore(computeSourceMergeReadiness(metrics))
	return metrics, nil
}

func (i DefaultInspector) inspectProvider(
	ctx context.Context,
	source SourceConfig,
	staleAfter time.Duration,
	now time.Time,
) (SCMSourceMetrics, bool, error) {
	switch strings.ToLower(strings.TrimSpace(source.Kind)) {
	case "github":
		if source.Repository == "" {
			return SCMSourceMetrics{}, true, fmt.Errorf("github repository is required")
		}
		return i.inspectGitHub(ctx, source, staleAfter, now)
	case "gitlab":
		if source.ProjectID == "" {
			return SCMSourceMetrics{}, true, fmt.Errorf("gitlab project_id is required")
		}
		return i.inspectGitLab(ctx, source, staleAfter, now)
	case "gitverse":
		return SCMSourceMetrics{}, false, nil
	default:
		return SCMSourceMetrics{}, false, nil
	}
}

func (i DefaultInspector) inspectGitHub(
	ctx context.Context,
	source SourceConfig,
	staleAfter time.Duration,
	now time.Time,
) (SCMSourceMetrics, bool, error) {
	var pulls []struct {
		Number    int    `json:"number"`
		Draft     bool   `json:"draft"`
		UpdatedAt string `json:"updated_at"`
		Head      struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	endpoint := fmt.Sprintf("/repos/%s/pulls?state=open&per_page=100", source.Repository)
	if err := i.getJSON(ctx, source, githubAPIBase(source), endpoint, &pulls); err != nil {
		return SCMSourceMetrics{}, true, fmt.Errorf("github provider metrics unavailable: %w", err)
	}

	metrics := SCMSourceMetrics{}
	for _, pull := range pulls {
		metrics.OpenChangeRequests++
		if updatedAt, err := time.Parse(time.RFC3339, pull.UpdatedAt); err == nil && now.Sub(updatedAt) > staleAfter {
			metrics.StaleChangeRequests++
		}
		if approved, err := i.githubApproved(ctx, source, pull.Number); err == nil && approved {
			metrics.ApprovedChangeRequests++
		}
		if failing, err := i.githubFailingChecks(ctx, source, pull.Head.SHA); err == nil && failing {
			metrics.FailingChangeRequests++
		}
	}
	return metrics, true, nil
}

func (i DefaultInspector) githubApproved(ctx context.Context, source SourceConfig, number int) (bool, error) {
	var reviews []struct {
		State string `json:"state"`
	}
	if err := i.getJSON(
		ctx,
		source,
		githubAPIBase(source),
		fmt.Sprintf("/repos/%s/pulls/%d/reviews?per_page=100", source.Repository, number),
		&reviews,
	); err != nil {
		return false, err
	}
	for _, review := range reviews {
		if strings.EqualFold(review.State, "APPROVED") {
			return true, nil
		}
	}
	return false, nil
}

func (i DefaultInspector) githubFailingChecks(ctx context.Context, source SourceConfig, sha string) (bool, error) {
	if strings.TrimSpace(sha) == "" {
		return false, nil
	}
	var payload struct {
		CheckRuns []struct {
			Conclusion string `json:"conclusion"`
		} `json:"check_runs"`
	}
	if err := i.getJSON(
		ctx,
		source,
		githubAPIBase(source),
		fmt.Sprintf("/repos/%s/commits/%s/check-runs?per_page=100", source.Repository, sha),
		&payload,
	); err != nil {
		return false, err
	}
	for _, run := range payload.CheckRuns {
		switch strings.ToLower(strings.TrimSpace(run.Conclusion)) {
		case "failure", "cancelled", "timed_out", "action_required":
			return true, nil
		}
	}
	return false, nil
}

func (i DefaultInspector) inspectGitLab(
	ctx context.Context,
	source SourceConfig,
	staleAfter time.Duration,
	now time.Time,
) (SCMSourceMetrics, bool, error) {
	var mergeRequests []struct {
		IID            int    `json:"iid"`
		UpdatedAt      string `json:"updated_at"`
		Draft          bool   `json:"draft"`
		WorkInProgress bool   `json:"work_in_progress"`
		HeadPipeline   *struct {
			Status string `json:"status"`
		} `json:"head_pipeline"`
	}
	endpoint := fmt.Sprintf("/projects/%s/merge_requests?state=opened&per_page=100", encodeProjectID(source.ProjectID))
	if err := i.getJSON(ctx, source, gitlabAPIBase(source), endpoint, &mergeRequests); err != nil {
		return SCMSourceMetrics{}, true, fmt.Errorf("gitlab provider metrics unavailable: %w", err)
	}

	metrics := SCMSourceMetrics{}
	for _, mr := range mergeRequests {
		metrics.OpenChangeRequests++
		if updatedAt, err := time.Parse(time.RFC3339, mr.UpdatedAt); err == nil && now.Sub(updatedAt) > staleAfter {
			metrics.StaleChangeRequests++
		}
		if mr.HeadPipeline != nil {
			switch strings.ToLower(strings.TrimSpace(mr.HeadPipeline.Status)) {
			case "failed", "canceled":
				metrics.FailingChangeRequests++
			}
		}
		approved, err := i.gitlabApproved(ctx, source, mr.IID)
		if err == nil && approved && !mr.Draft && !mr.WorkInProgress {
			metrics.ApprovedChangeRequests++
		}
	}
	return metrics, true, nil
}

func (i DefaultInspector) gitlabApproved(ctx context.Context, source SourceConfig, iid int) (bool, error) {
	var payload struct {
		Approved bool `json:"approved"`
	}
	endpoint := fmt.Sprintf("/projects/%s/merge_requests/%d/approvals", encodeProjectID(source.ProjectID), iid)
	if err := i.getJSON(ctx, source, gitlabAPIBase(source), endpoint, &payload); err != nil {
		return false, err
	}
	return payload.Approved, nil
}

func (i DefaultInspector) getJSON(
	ctx context.Context,
	source SourceConfig,
	baseURL string,
	endpoint string,
	out any,
) error {
	client := i.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinURL(baseURL, endpoint), nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(source.APIToken) != "" {
		switch strings.ToLower(strings.TrimSpace(source.Kind)) {
		case "gitlab":
			req.Header.Set("PRIVATE-TOKEN", source.APIToken)
		default:
			req.Header.Set("Authorization", "Bearer "+source.APIToken)
			req.Header.Set("Accept", "application/vnd.github+json")
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func githubAPIBase(source SourceConfig) string {
	if strings.TrimSpace(source.APIURL) != "" {
		return strings.TrimRight(source.APIURL, "/")
	}
	return "https://api.github.com"
}

func gitlabAPIBase(source SourceConfig) string {
	if strings.TrimSpace(source.APIURL) != "" {
		return strings.TrimRight(source.APIURL, "/")
	}
	return "https://gitlab.com/api/v4"
}

func joinURL(baseURL string, endpoint string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + endpoint
	}
	target, err := url.Parse(endpoint)
	if err != nil {
		return strings.TrimRight(baseURL, "/") + endpoint
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(target.Path, "/")
	base.RawQuery = target.RawQuery
	return base.String()
}

func mergeSCMMetrics(base SCMSourceMetrics, extra SCMSourceMetrics) SCMSourceMetrics {
	base.Kind = firstNonEmpty(base.Kind, extra.Kind)
	base.Name = firstNonEmpty(base.Name, extra.Name)
	base.RepoPath = firstNonEmpty(base.RepoPath, extra.RepoPath)
	base.MainBranch = firstNonEmpty(base.MainBranch, extra.MainBranch)
	base.Repository = firstNonEmpty(base.Repository, extra.Repository)
	base.ProjectID = firstNonEmpty(base.ProjectID, extra.ProjectID)
	base.Branches += extra.Branches
	base.UnmergedBranches += extra.UnmergedBranches
	base.StaleBranches += extra.StaleBranches
	base.DriftCommits += extra.DriftCommits
	base.AheadCommits += extra.AheadCommits
	base.OpenChangeRequests += extra.OpenChangeRequests
	base.ApprovedChangeRequests += extra.ApprovedChangeRequests
	base.FailingChangeRequests += extra.FailingChangeRequests
	base.StaleChangeRequests += extra.StaleChangeRequests
	if extra.MaxAgeHours > base.MaxAgeHours {
		base.MaxAgeHours = extra.MaxAgeHours
	}
	base.Warnings = append(base.Warnings, extra.Warnings...)
	return base
}

func computeSourceMergeReadiness(metrics SCMSourceMetrics) float64 {
	branchBase := maxInt(metrics.Branches, 1)
	branchComponent := 0.40*(1-ratio(metrics.DriftCommits, maxInt(branchBase*8, 1))) +
		0.35*(1-ratio(metrics.StaleBranches, branchBase)) +
		0.25*(1-ratio(metrics.UnmergedBranches, branchBase))
	changeBase := maxInt(metrics.OpenChangeRequests, 1)
	reviewComponent := 1.0
	if metrics.OpenChangeRequests > 0 {
		reviewComponent =
			0.40*(1-ratio(metrics.FailingChangeRequests, changeBase)) +
				0.35*ratio(metrics.ApprovedChangeRequests, changeBase) +
				0.25*(1-ratio(metrics.StaleChangeRequests, changeBase))
	}
	return 100 * (0.55*branchComponent + 0.45*reviewComponent)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func encodeProjectID(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.Contains(strings.ToLower(trimmed), "%2f") {
		return trimmed
	}
	return url.PathEscape(trimmed)
}
