package insights

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
	"github.com/alexey-pronkin/symphony-go/arpego/internal/tracker"
)

type TaskProvider interface {
	ListTasks(context.Context) ([]tracker.Issue, error)
}

type RuntimeProvider interface {
	Snapshot() orchestrator.Snapshot
}

type GitInspector interface {
	Inspect(context.Context, SourceConfig, time.Duration, time.Time) (SCMSourceMetrics, error)
}

type Service struct {
	tasks            TaskProvider
	runtime          RuntimeProvider
	inspector        GitInspector
	sources          []SourceConfig
	now              func() time.Time
	staleAfter       time.Duration
	throughputWindow time.Duration
}

type Options struct {
	Tasks            TaskProvider
	Runtime          RuntimeProvider
	Inspector        GitInspector
	Sources          []SourceConfig
	Now              func() time.Time
	StaleAfter       time.Duration
	ThroughputWindow time.Duration
}

func NewService(opts Options) *Service {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	inspector := opts.Inspector
	if inspector == nil {
		inspector = GitGoInspector{}
	}
	staleAfter := opts.StaleAfter
	if staleAfter <= 0 {
		staleAfter = 72 * time.Hour
	}
	throughputWindow := opts.ThroughputWindow
	if throughputWindow <= 0 {
		throughputWindow = 7 * 24 * time.Hour
	}
	return &Service{
		tasks:            opts.Tasks,
		runtime:          opts.Runtime,
		inspector:        inspector,
		sources:          append([]SourceConfig(nil), opts.Sources...),
		now:              now,
		staleAfter:       staleAfter,
		throughputWindow: throughputWindow,
	}
}

func (s *Service) Delivery(ctx context.Context) (DeliveryReport, error) {
	now := s.now().UTC()
	report := DeliveryReport{
		GeneratedAt: now,
		Warnings:    []string{},
	}

	snapshot := orchestrator.Snapshot{}
	if s.runtime != nil {
		snapshot = s.runtime.Snapshot()
	}

	trackerMetrics, trackerWarnings := s.buildTrackerMetrics(ctx, snapshot, now)
	report.Tracker = trackerMetrics
	report.Warnings = append(report.Warnings, trackerWarnings...)

	scmMetrics, scmWarnings := s.buildSCMMetrics(ctx, now)
	report.SCM = scmMetrics
	report.Warnings = append(report.Warnings, scmWarnings...)

	report.Summary = buildSummary(report.Tracker, report.SCM)
	return report, nil
}

func (s *Service) buildTrackerMetrics(
	ctx context.Context,
	snapshot orchestrator.Snapshot,
	now time.Time,
) (TrackerMetrics, []string) {
	metrics := TrackerMetrics{
		Runtime: RuntimeMetrics{
			RunningSessions:  snapshot.Counts.Running,
			RetryingSessions: snapshot.Counts.Retrying,
			ActiveTokens:     snapshot.CodexTotals.TotalTokens,
		},
	}
	if s.tasks == nil {
		return metrics, []string{"task metrics unavailable: task platform is not configured"}
	}

	tasks, err := s.tasks.ListTasks(ctx)
	if err != nil {
		return metrics, []string{fmt.Sprintf("task metrics unavailable: %v", err)}
	}

	windowStart := now.Add(-s.throughputWindow)
	var ageHoursTotal float64
	var activeForAge int
	for _, issue := range tasks {
		metrics.TotalTasks++
		state := normalizeState(issue.State)
		if isActiveState(state) {
			metrics.ActiveTasks++
			if issue.CreatedAt != nil {
				ageHoursTotal += now.Sub(*issue.CreatedAt).Hours()
				activeForAge++
			}
		}
		if strings.Contains(state, "review") {
			metrics.ReviewTasks++
		}
		if isBlocked(issue) {
			metrics.BlockedTasks++
		}
		if state == "done" && issue.UpdatedAt != nil && !issue.UpdatedAt.Before(windowStart) {
			metrics.DoneLastWindow++
		}
	}
	if activeForAge > 0 {
		metrics.AvgActiveAgeHours = round2(ageHoursTotal / float64(activeForAge))
	}

	metrics.Agile = AgileMetrics{
		ThroughputLastWindow: metrics.DoneLastWindow,
		CompletionRatio:      ratio(metrics.DoneLastWindow, maxInt(metrics.TotalTasks, 1)),
		ReviewLoad:           ratio(metrics.ReviewTasks, maxInt(metrics.ActiveTasks, 1)),
	}

	wipCount := countWIP(tasks)
	metrics.Kanban = KanbanMetrics{
		WIPCount:       wipCount,
		BlockedRatio:   ratio(metrics.BlockedTasks, maxInt(metrics.ActiveTasks, 1)),
		AgingWorkRatio: clamp01(metrics.AvgActiveAgeHours / 168.0),
		FlowLoad:       ratio(wipCount+metrics.ReviewTasks, maxInt(metrics.TotalTasks, 1)),
	}
	metrics.BacklogPressure = round2(ratio(countState(tasks, "todo"), maxInt(metrics.DoneLastWindow, 1)))
	return metrics, nil
}

func (s *Service) buildSCMMetrics(ctx context.Context, now time.Time) (SCMMetrics, []string) {
	out := SCMMetrics{
		Sources: []SCMSourceMetrics{},
	}
	if len(s.sources) == 0 {
		return out, []string{"scm metrics degraded: no SCM sources configured"}
	}

	warnings := []string{}
	for _, source := range s.sources {
		metrics, err := s.inspector.Inspect(ctx, source, s.staleAfter, now)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("scm source %s unavailable: %v", source.Name, err))
			out.Sources = append(out.Sources, SCMSourceMetrics{
				Kind:       source.Kind,
				Name:       source.Name,
				RepoPath:   source.RepoPath,
				MainBranch: source.MainBranch,
				Warnings:   []string{err.Error()},
			})
			continue
		}
		out.ActiveSources++
		out.Sources = append(out.Sources, metrics)
		out.Totals.Branches += metrics.Branches
		out.Totals.UnmergedBranches += metrics.UnmergedBranches
		out.Totals.StaleBranches += metrics.StaleBranches
		out.Totals.DriftCommits += metrics.DriftCommits
		out.Totals.AheadCommits += metrics.AheadCommits
		if metrics.MaxAgeHours > out.Totals.MaxAgeHours {
			out.Totals.MaxAgeHours = metrics.MaxAgeHours
		}
	}
	return out, warnings
}

func buildSummary(trackerMetrics TrackerMetrics, scmMetrics SCMMetrics) DeliverySummary {
	flowScore := clampScore(
		100 *
			(0.45*clamp01(trackerMetrics.Agile.CompletionRatio*2.5) +
				0.30*(1-trackerMetrics.Kanban.BlockedRatio) +
				0.25*(1-clamp01(trackerMetrics.Agile.ReviewLoad))),
	)
	mergeScore := 100
	if scmMetrics.ActiveSources > 0 {
		branchBase := maxInt(scmMetrics.Totals.Branches, 1)
		mergeScore = clampScore(
			100 *
				(0.40*(1-ratio(scmMetrics.Totals.DriftCommits, maxInt(branchBase*8, 1))) +
					0.35*(1-ratio(scmMetrics.Totals.StaleBranches, branchBase)) +
					0.25*(1-ratio(scmMetrics.Totals.UnmergedBranches, branchBase))),
		)
	}
	predictabilityScore := clampScore(
		100 *
			(0.55*clamp01(trackerMetrics.Agile.CompletionRatio*3.0) +
				0.25*(1-clamp01(trackerMetrics.BacklogPressure/6.0)) +
				0.20*(1-ratio(trackerMetrics.Runtime.RetryingSessions, maxInt(trackerMetrics.Runtime.RunningSessions+1, 1)))),
	)
	deliveryHealth := clampScore(float64(flowScore)*0.45 + float64(mergeScore)*0.35 + float64(predictabilityScore)*0.20)

	return DeliverySummary{
		DeliveryHealth: metricCard(
			"delivery_health",
			"Delivery health",
			deliveryHealth,
			fmt.Sprintf(
				"%d active tasks, %d blocked, %d retrying sessions.",
				trackerMetrics.ActiveTasks,
				trackerMetrics.BlockedTasks,
				trackerMetrics.Runtime.RetryingSessions,
			),
		),
		FlowEfficiency: metricCard(
			"flow_efficiency",
			"Flow efficiency",
			flowScore,
			fmt.Sprintf(
				"%d completed in window, review load %.0f%%.",
				trackerMetrics.DoneLastWindow,
				trackerMetrics.Agile.ReviewLoad*100,
			),
		),
		MergeReadiness: metricCard(
			"merge_readiness",
			"Merge readiness",
			mergeScore,
			fmt.Sprintf(
				"%d unmerged branches, %d drift commits.",
				scmMetrics.Totals.UnmergedBranches,
				scmMetrics.Totals.DriftCommits,
			),
		),
		Predictability: metricCard(
			"predictability",
			"Predictability",
			predictabilityScore,
			fmt.Sprintf(
				"completion ratio %.0f%%, backlog pressure %.1fx.",
				trackerMetrics.Agile.CompletionRatio*100,
				trackerMetrics.BacklogPressure,
			),
		),
	}
}

func metricCard(key, label string, score int, detail string) IntegralMetric {
	return IntegralMetric{
		Key:    key,
		Label:  label,
		Score:  score,
		Status: scoreStatus(score),
		Detail: detail,
	}
}

func scoreStatus(score int) string {
	switch {
	case score >= 80:
		return "strong"
	case score >= 60:
		return "watch"
	default:
		return "risk"
	}
}

func normalizeState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func isActiveState(state string) bool {
	switch state {
	case "todo", "in progress", "review":
		return true
	default:
		return false
	}
}

func isBlocked(issue tracker.Issue) bool {
	for _, blocker := range issue.BlockedBy {
		if blocker.Identifier == nil {
			continue
		}
		if blocker.State == nil || !isTerminalState(normalizeState(*blocker.State)) {
			return true
		}
	}
	return false
}

func isTerminalState(state string) bool {
	switch state {
	case "done", "closed", "cancelled", "canceled", "duplicate":
		return true
	default:
		return false
	}
}

func countWIP(tasks []tracker.Issue) int {
	total := 0
	for _, issue := range tasks {
		state := normalizeState(issue.State)
		if state == "in progress" || state == "review" {
			total++
		}
	}
	return total
}

func countState(tasks []tracker.Issue, target string) int {
	total := 0
	for _, issue := range tasks {
		if normalizeState(issue.State) == target {
			total++
		}
	}
	return total
}

func ratio(a, b int) float64 {
	if b <= 0 {
		return 0
	}
	return round2(float64(a) / float64(b))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func clampScore(value float64) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return int(math.Round(value))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
