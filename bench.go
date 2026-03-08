package bench

import (
	"strings"
	"time"
)

// Result captures the outcome of a single agent run on a task.
type Result struct {
	Agent     string       `json:"agent"`
	Task      string       `json:"task"`
	Repo      string       `json:"repo"`
	Commit    string       `json:"commit"`
	Trial     int          `json:"trial"`
	Timestamp time.Time    `json:"timestamp"`
	Metrics   Metrics      `json:"metrics"`
	Git       GitStats     `json:"git"`
	Quality   QualityCheck `json:"quality"`
	Error     string       `json:"error,omitempty"`
}

// Metrics tracks token usage and timing.
type Metrics struct {
	InputTokens     int           `json:"input_tokens"`
	OutputTokens    int           `json:"output_tokens"`
	TotalTokens     int           `json:"total_tokens"`
	CacheReadTokens int           `json:"cache_read_tokens,omitempty"`
	CostUSD         float64       `json:"cost_usd"`
	Turns           int           `json:"turns"`
	ToolCalls       int           `json:"tool_calls"`
	Model           string        `json:"model,omitempty"`
	WallTime        time.Duration `json:"wall_time_ns"`
	WallTimeSec     float64       `json:"wall_time_sec"`
}

// GitStats captures the code diff.
type GitStats struct {
	FilesChanged int      `json:"files_changed"`
	LinesAdded   int      `json:"lines_added"`
	LinesRemoved int      `json:"lines_removed"`
	ChangedFiles []string `json:"changed_files"`
}

// QualityCheck captures build/test results.
type QualityCheck struct {
	Builds     bool   `json:"builds"`
	TestsPass  bool   `json:"tests_pass"`
	TestOutput string `json:"test_output,omitempty"`
	ScopeCreep int    `json:"scope_creep"`
}

// ModelPricing holds per-million-token prices.
type ModelPricing struct {
	InputPerM     float64
	OutputPerM    float64
	CacheReadPerM float64
}

// Known model prices (USD per million tokens).
// Keys are prefix-matched: "claude-sonnet-4" matches "claude-sonnet-4-5-20250929".
var Pricing = map[string]ModelPricing{
	"claude-sonnet-4": {InputPerM: 3.0, OutputPerM: 15.0, CacheReadPerM: 0.30},
	"claude-opus-4":   {InputPerM: 15.0, OutputPerM: 75.0, CacheReadPerM: 3.75},
	"claude-haiku-3":  {InputPerM: 0.80, OutputPerM: 4.0, CacheReadPerM: 0.08},
	"glm-5":           {InputPerM: 0.50, OutputPerM: 2.0, CacheReadPerM: 0.10},
	"glm-4":           {InputPerM: 0.50, OutputPerM: 2.0, CacheReadPerM: 0.10},
}

// CalculateCost computes USD cost from token counts and model.
func (m *Metrics) CalculateCost() {
	pricing, ok := Pricing[m.Model]
	if !ok {
		// Try prefix match (longest prefix wins)
		bestLen := 0
		for k, v := range Pricing {
			if strings.HasPrefix(m.Model, k) && len(k) > bestLen {
				pricing = v
				ok = true
				bestLen = len(k)
			}
		}
	}
	if !ok {
		return
	}
	m.CostUSD = float64(m.InputTokens)/1_000_000*pricing.InputPerM +
		float64(m.OutputTokens)/1_000_000*pricing.OutputPerM +
		float64(m.CacheReadTokens)/1_000_000*pricing.CacheReadPerM
}

// AgentSummary holds averaged results across multiple trials for one agent.
type AgentSummary struct {
	Agent        string
	Model        string
	Trials       int
	Successes    int // trials where build+test passed
	AvgInput     float64
	AvgOutput    float64
	AvgTotal     float64
	AvgCache     float64
	AvgCost      float64
	AvgTime      float64
	AvgTools     float64
	AvgFiles     float64
	AvgAdded     float64
	AvgRemoved   float64
	Results      []Result
}

// Summarize groups results by agent and computes averages.
func Summarize(results []Result) []AgentSummary {
	groups := map[string]*AgentSummary{}

	for _, r := range results {
		s, ok := groups[r.Agent]
		if !ok {
			s = &AgentSummary{Agent: r.Agent, Model: r.Metrics.Model}
			groups[r.Agent] = s
		}
		s.Trials++
		if r.Quality.Builds && r.Quality.TestsPass && r.Error == "" {
			s.Successes++
		}
		s.AvgInput += float64(r.Metrics.InputTokens)
		s.AvgOutput += float64(r.Metrics.OutputTokens)
		s.AvgTotal += float64(r.Metrics.TotalTokens)
		s.AvgCache += float64(r.Metrics.CacheReadTokens)
		s.AvgCost += r.Metrics.CostUSD
		s.AvgTime += r.Metrics.WallTimeSec
		s.AvgTools += float64(r.Metrics.ToolCalls)
		s.AvgFiles += float64(r.Git.FilesChanged)
		s.AvgAdded += float64(r.Git.LinesAdded)
		s.AvgRemoved += float64(r.Git.LinesRemoved)
		s.Results = append(s.Results, r)
	}

	var out []AgentSummary
	for _, s := range groups {
		n := float64(s.Trials)
		s.AvgInput /= n
		s.AvgOutput /= n
		s.AvgTotal /= n
		s.AvgCache /= n
		s.AvgCost /= n
		s.AvgTime /= n
		s.AvgTools /= n
		s.AvgFiles /= n
		s.AvgAdded /= n
		s.AvgRemoved /= n
		out = append(out, *s)
	}

	return out
}
