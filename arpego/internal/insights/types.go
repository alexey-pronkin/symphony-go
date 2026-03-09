package insights

import "time"

type DeliveryReport struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Summary     DeliverySummary `json:"summary"`
	Tracker     TrackerMetrics  `json:"tracker"`
	SCM         SCMMetrics      `json:"scm"`
	Warnings    []string        `json:"warnings"`
}

type DeliveryTrendQuery struct {
	Window string
	Limit  int
}

type DeliveryTrendReport struct {
	GeneratedAt time.Time            `json:"generated_at"`
	Window      string               `json:"window"`
	Limit       int                  `json:"limit"`
	Available   bool                 `json:"available"`
	Points      []DeliveryTrendPoint `json:"points"`
	Rollups     DeliveryTrendRollups `json:"rollups"`
	Alerts      []DeliveryTrendAlert `json:"alerts"`
	Warnings    []string             `json:"warnings"`
}

type DeliveryTrendPoint struct {
	CapturedAt          time.Time `json:"captured_at"`
	DeliveryHealth      int       `json:"delivery_health"`
	FlowEfficiency      int       `json:"flow_efficiency"`
	MergeReadiness      int       `json:"merge_readiness"`
	Predictability      int       `json:"predictability"`
	ActiveTasks         int       `json:"active_tasks"`
	BlockedTasks        int       `json:"blocked_tasks"`
	DoneLastWindow      int       `json:"done_last_window"`
	WIPCount            int       `json:"wip_count"`
	OpenChangeRequests  int       `json:"open_change_requests"`
	FailingChangeChecks int       `json:"failing_change_checks"`
	WarningCount        int       `json:"warning_count"`
}

type DeliveryTrendRollups struct {
	HealthAverage       int     `json:"health_average"`
	HealthDelta         int     `json:"health_delta"`
	HealthSlope         float64 `json:"health_slope"`
	FlowAverage         int     `json:"flow_average"`
	MergeAverage        int     `json:"merge_average"`
	PredictabilityTrend int     `json:"predictability_trend"`
	WarningPressure     float64 `json:"warning_pressure"`
	InsufficientSamples bool    `json:"insufficient_samples"`
}

type DeliveryTrendAlert struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

type DeliverySummary struct {
	DeliveryHealth IntegralMetric `json:"delivery_health"`
	FlowEfficiency IntegralMetric `json:"flow_efficiency"`
	MergeReadiness IntegralMetric `json:"merge_readiness"`
	Predictability IntegralMetric `json:"predictability"`
}

type IntegralMetric struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Score  int    `json:"score"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type TrackerMetrics struct {
	TotalTasks        int            `json:"total_tasks"`
	ActiveTasks       int            `json:"active_tasks"`
	BlockedTasks      int            `json:"blocked_tasks"`
	ReviewTasks       int            `json:"review_tasks"`
	DoneLastWindow    int            `json:"done_last_window"`
	AvgActiveAgeHours float64        `json:"avg_active_age_hours"`
	BacklogPressure   float64        `json:"backlog_pressure"`
	Runtime           RuntimeMetrics `json:"runtime"`
	Agile             AgileMetrics   `json:"agile"`
	Kanban            KanbanMetrics  `json:"kanban"`
}

type RuntimeMetrics struct {
	RunningSessions  int `json:"running_sessions"`
	RetryingSessions int `json:"retrying_sessions"`
	ActiveTokens     int `json:"active_tokens"`
}

type AgileMetrics struct {
	ThroughputLastWindow int     `json:"throughput_last_window"`
	CompletionRatio      float64 `json:"completion_ratio"`
	ReviewLoad           float64 `json:"review_load"`
}

type KanbanMetrics struct {
	WIPCount       int     `json:"wip_count"`
	BlockedRatio   float64 `json:"blocked_ratio"`
	AgingWorkRatio float64 `json:"aging_work_ratio"`
	FlowLoad       float64 `json:"flow_load"`
}

type SCMMetrics struct {
	ActiveSources int                `json:"active_sources"`
	Sources       []SCMSourceMetrics `json:"sources"`
	Totals        SCMTotals          `json:"totals"`
}

type SCMTotals struct {
	Branches               int     `json:"branches"`
	UnmergedBranches       int     `json:"unmerged_branches"`
	StaleBranches          int     `json:"stale_branches"`
	DriftCommits           int     `json:"drift_commits"`
	AheadCommits           int     `json:"ahead_commits"`
	MaxAgeHours            float64 `json:"max_age_hours"`
	OpenChangeRequests     int     `json:"open_change_requests"`
	ApprovedChangeRequests int     `json:"approved_change_requests"`
	FailingChangeRequests  int     `json:"failing_change_requests"`
	StaleChangeRequests    int     `json:"stale_change_requests"`
}

type SCMSourceMetrics struct {
	Kind                   string   `json:"kind"`
	Name                   string   `json:"name"`
	RepoPath               string   `json:"repo_path"`
	MainBranch             string   `json:"main_branch"`
	Repository             string   `json:"repository,omitempty"`
	ProjectID              string   `json:"project_id,omitempty"`
	Branches               int      `json:"branches"`
	UnmergedBranches       int      `json:"unmerged_branches"`
	StaleBranches          int      `json:"stale_branches"`
	DriftCommits           int      `json:"drift_commits"`
	AheadCommits           int      `json:"ahead_commits"`
	MaxAgeHours            float64  `json:"max_age_hours"`
	OpenChangeRequests     int      `json:"open_change_requests"`
	ApprovedChangeRequests int      `json:"approved_change_requests"`
	FailingChangeRequests  int      `json:"failing_change_requests"`
	StaleChangeRequests    int      `json:"stale_change_requests"`
	MergeReadiness         int      `json:"merge_readiness"`
	Warnings               []string `json:"warnings,omitempty"`
}

type SourceConfig struct {
	Kind       string
	Name       string
	RepoPath   string
	MainBranch string
	APIURL     string
	Repository string
	ProjectID  string
	APIToken   string
}
