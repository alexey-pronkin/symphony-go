package securityscan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexey-pronkin/symphony-go/arpego/internal/orchestrator"
)

type WorkspaceScanner interface {
	ScanWorkspace(context.Context, string) orchestrator.WorkspaceScan
}

type TrivyScanner struct {
	command string
	timeout time.Duration
	ttl     time.Duration

	mu    sync.Mutex
	cache map[string]cachedScan
}

type cachedScan struct {
	expiresAt time.Time
	result    orchestrator.WorkspaceScan
}

func NewTrivyScanner(command string, timeout, ttl time.Duration) *TrivyScanner {
	if strings.TrimSpace(command) == "" {
		command = "trivy"
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &TrivyScanner{
		command: command,
		timeout: timeout,
		ttl:     ttl,
		cache:   map[string]cachedScan{},
	}
}

func (s *TrivyScanner) ScanWorkspace(ctx context.Context, workspacePath string) orchestrator.WorkspaceScan {
	if strings.TrimSpace(workspacePath) == "" {
		return unavailableScan("workspace path is empty")
	}
	if _, err := os.Stat(workspacePath); err != nil {
		return unavailableScan(err.Error())
	}

	now := time.Now().UTC()
	s.mu.Lock()
	if cached, ok := s.cache[workspacePath]; ok && now.Before(cached.expiresAt) {
		s.mu.Unlock()
		return cached.result
	}
	s.mu.Unlock()

	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	started := time.Now()
	cmd := exec.CommandContext(
		scanCtx,
		s.command,
		"fs",
		"--quiet",
		"--format",
		"json",
		"--severity",
		"HIGH,CRITICAL",
		"--scanners",
		"vuln,misconfig,secret",
		workspacePath,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.Bytes()
	duration := time.Since(started)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(output) > 0 {
			// Trivy returns non-zero when findings are present, but still emits JSON to stdout.
		} else {
			message := strings.TrimSpace(stderr.String())
			if message == "" {
				message = err.Error()
			}
			result := errorScan(message, duration)
			s.store(workspacePath, result, now)
			return result
		}
	}

	result, parseErr := parseTrivyOutput(output, duration)
	if parseErr != nil {
		result = errorScan(parseErr.Error(), duration)
	} else {
		scannedAt := now
		result.ScannedAt = &scannedAt
	}
	s.store(workspacePath, result, now)
	return result
}

func (s *TrivyScanner) store(workspacePath string, result orchestrator.WorkspaceScan, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[workspacePath] = cachedScan{
		expiresAt: now.Add(s.ttl),
		result:    result,
	}
}

type trivyReport struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Target            string               `json:"Target"`
	Vulnerabilities   []trivyVulnerability `json:"Vulnerabilities"`
	Misconfigurations []trivyMisconfig     `json:"Misconfigurations"`
	Secrets           []trivySecret        `json:"Secrets"`
}

type trivyVulnerability struct {
	ID               string `json:"VulnerabilityID"`
	Title            string `json:"Title"`
	Severity         string `json:"Severity"`
	PrimaryURL       string `json:"PrimaryURL"`
	PackageName      string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
}

type trivyMisconfig struct {
	ID         string `json:"ID"`
	Title      string `json:"Title"`
	Severity   string `json:"Severity"`
	Message    string `json:"Message"`
	PrimaryURL string `json:"PrimaryURL"`
}

type trivySecret struct {
	RuleID   string `json:"RuleID"`
	Title    string `json:"Title"`
	Severity string `json:"Severity"`
}

func parseTrivyOutput(raw []byte, duration time.Duration) (orchestrator.WorkspaceScan, error) {
	report := trivyReport{}
	if err := json.Unmarshal(raw, &report); err != nil {
		return orchestrator.WorkspaceScan{}, err
	}

	findings := make([]orchestrator.WorkspaceScanFinding, 0)
	summary := orchestrator.WorkspaceScanSummary{}

	for _, result := range report.Results {
		for _, vuln := range result.Vulnerabilities {
			finding := orchestrator.WorkspaceScanFinding{
				ID:               vuln.ID,
				Category:         "vulnerability",
				Severity:         normalizeSeverity(vuln.Severity),
				Title:            firstNonEmpty(vuln.Title, vuln.ID),
				Target:           result.Target,
				PrimaryURL:       stringPointer(vuln.PrimaryURL),
				PackageName:      vuln.PackageName,
				InstalledVersion: vuln.InstalledVersion,
				FixedVersion:     vuln.FixedVersion,
			}
			findings = append(findings, finding)
			summary.Total++
			summary.Vulnerabilities++
			incrementSeverity(&summary, finding.Severity)
		}
		for _, misconfig := range result.Misconfigurations {
			finding := orchestrator.WorkspaceScanFinding{
				ID:         misconfig.ID,
				Category:   "misconfiguration",
				Severity:   normalizeSeverity(misconfig.Severity),
				Title:      firstNonEmpty(misconfig.Title, misconfig.Message, misconfig.ID),
				Target:     result.Target,
				PrimaryURL: stringPointer(misconfig.PrimaryURL),
			}
			findings = append(findings, finding)
			summary.Total++
			summary.Misconfigs++
			incrementSeverity(&summary, finding.Severity)
		}
		for _, secret := range result.Secrets {
			finding := orchestrator.WorkspaceScanFinding{
				ID:       secret.RuleID,
				Category: "secret",
				Severity: normalizeSeverity(secret.Severity),
				Title:    firstNonEmpty(secret.Title, secret.RuleID),
				Target:   result.Target,
			}
			findings = append(findings, finding)
			summary.Total++
			summary.Secrets++
			incrementSeverity(&summary, finding.Severity)
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		left := severityRank(findings[i].Severity)
		right := severityRank(findings[j].Severity)
		if left != right {
			return left < right
		}
		if findings[i].Category != findings[j].Category {
			return findings[i].Category < findings[j].Category
		}
		return findings[i].Title < findings[j].Title
	})
	if len(findings) > 8 {
		findings = findings[:8]
	}

	status := "ok"
	if summary.Total > 0 {
		status = "findings"
	}
	return orchestrator.WorkspaceScan{
		Status:     status,
		DurationMs: duration.Milliseconds(),
		Summary:    summary,
		Findings:   findings,
	}, nil
}

func incrementSeverity(summary *orchestrator.WorkspaceScanSummary, severity string) {
	switch severity {
	case "critical":
		summary.Critical++
	case "high":
		summary.High++
	}
}

func unavailableScan(message string) orchestrator.WorkspaceScan {
	return orchestrator.WorkspaceScan{
		Status: "unavailable",
		Error:  stringPointer(message),
	}
}

func errorScan(message string, duration time.Duration) orchestrator.WorkspaceScan {
	return orchestrator.WorkspaceScan{
		Status:     "error",
		DurationMs: duration.Milliseconds(),
		Error:      stringPointer(message),
	}
}

func normalizeSeverity(severity string) string {
	return strings.ToLower(strings.TrimSpace(severity))
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "high":
		return 1
	default:
		return 2
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func stringPointer(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
