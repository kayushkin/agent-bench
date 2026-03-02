package bench

import "time"

// Result captures the outcome of a single agent run on a task.
type Result struct {
	Agent     string        `json:"agent"`
	Task      string        `json:"task"`
	Repo      string        `json:"repo"`
	Commit    string        `json:"commit"`
	Timestamp time.Time     `json:"timestamp"`
	Metrics   Metrics       `json:"metrics"`
	Git       GitStats      `json:"git"`
	Quality   QualityCheck  `json:"quality"`
	Error     string        `json:"error,omitempty"`
}

// Metrics tracks token usage and timing.
type Metrics struct {
	InputTokens  int           `json:"input_tokens"`
	OutputTokens int           `json:"output_tokens"`
	TotalTokens  int           `json:"total_tokens"`
	CostUSD      float64       `json:"cost_usd"`
	Turns        int           `json:"turns"`
	ToolCalls    int           `json:"tool_calls"`
	WallTime     time.Duration `json:"wall_time_ns"`
	WallTimeSec  float64       `json:"wall_time_sec"`
}

// GitStats captures the code diff.
type GitStats struct {
	FilesChanged  int      `json:"files_changed"`
	LinesAdded    int      `json:"lines_added"`
	LinesRemoved  int      `json:"lines_removed"`
	ChangedFiles  []string `json:"changed_files"`
}

// QualityCheck captures build/test results.
type QualityCheck struct {
	Builds     bool   `json:"builds"`
	TestsPass  bool   `json:"tests_pass"`
	TestOutput string `json:"test_output,omitempty"`
	ScopeCreep int    `json:"scope_creep"` // files changed outside expected scope
}
