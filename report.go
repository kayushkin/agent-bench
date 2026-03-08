package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Report generates a comparison from multiple results.
type Report struct {
	Results []Result `json:"results"`
}

// LoadResults reads all JSON result files from a directory.
func LoadResults(dir string) (*Report, error) {
	r := &Report{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var result Result
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}
		r.Results = append(r.Results, result)
	}

	return r, nil
}

// SaveResult writes a result to a JSON file.
func SaveResult(dir string, r *Result) error {
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	filename := r.Agent + ".json"
	if r.Trial > 0 {
		filename = fmt.Sprintf("%s-trial%d.json", r.Agent, r.Trial)
	}
	path := filepath.Join(dir, filename)
	return os.WriteFile(path, data, 0644)
}

func fmtTokens(n float64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", n/1_000)
	}
	return fmt.Sprintf("%.0f", n)
}

func fmtCost(c float64) string {
	if c < 0.001 {
		return fmt.Sprintf("$%.5f", c)
	}
	if c < 0.01 {
		return fmt.Sprintf("$%.4f", c)
	}
	return fmt.Sprintf("$%.3f", c)
}

// PrintComparison outputs a side-by-side comparison.
func (r *Report) PrintComparison() {
	if len(r.Results) == 0 {
		fmt.Println("No results to compare.")
		return
	}

	task := r.Results[0].Task
	summaries := Summarize(r.Results)

	// Sort by avg total tokens ascending
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].AvgTotal < summaries[j].AvgTotal
	})

	hasMultipleTrials := summaries[0].Trials > 1
	trialLabel := ""
	if hasMultipleTrials {
		trialLabel = fmt.Sprintf(" (%d trials avg)", summaries[0].Trials)
	}

	fmt.Printf("\n═══ Agent Benchmark: %s%s ═══\n\n", task, trialLabel)

	// Token & cost table
	fmt.Printf("%-12s %-10s %8s %8s %8s %10s %10s %8s %5s %5s %5s  %s\n",
		"Agent", "Model", "Input", "Output", "Total", "CacheRead", "CacheWrite", "Cost", "Turns", "Tools", "Time", "Pass")
	fmt.Println(strings.Repeat("─", 115))

	for _, s := range summaries {
		cacheReadStr := "-"
		if s.AvgCacheRead > 0 {
			cacheReadStr = fmtTokens(s.AvgCacheRead)
		}
		cacheCreateStr := "-"
		if s.AvgCacheCreate > 0 {
			cacheCreateStr = fmtTokens(s.AvgCacheCreate)
		}

		modelStr := s.Model
		if len(modelStr) > 10 {
			// Shorten model name
			modelStr = strings.ReplaceAll(modelStr, "claude-", "")
			modelStr = strings.ReplaceAll(modelStr, "-20250514", "")
			modelStr = strings.ReplaceAll(modelStr, "-20250", "")
		}

		passStr := fmt.Sprintf("%d/%d", s.Successes, s.Trials)

		fmt.Printf("%-12s %-10s %8s %8s %8s %10s %10s %8s %5.1f %5.0f %4.0fs  %s\n",
			s.Agent,
			modelStr,
			fmtTokens(s.AvgInput),
			fmtTokens(s.AvgOutput),
			fmtTokens(s.AvgTotal),
			cacheReadStr,
			cacheCreateStr,
			fmtCost(s.AvgCost),
			s.AvgTurns,
			s.AvgTools,
			s.AvgTime,
			passStr,
		)
	}

	fmt.Println()

	// Git diff comparison
	fmt.Printf("%-12s %8s %8s %8s\n", "Agent", "Files", "Added", "Removed")
	fmt.Println(strings.Repeat("─", 45))
	for _, s := range summaries {
		fmt.Printf("%-12s %8.0f %8.0f %8.0f\n",
			s.Agent,
			s.AvgFiles,
			s.AvgAdded,
			s.AvgRemoved,
		)
	}

	// Winner summary
	fmt.Println()
	if len(summaries) > 1 {
		best := summaries[0]
		worst := summaries[len(summaries)-1]

		if worst.AvgTotal > 0 {
			savings := (worst.AvgTotal - best.AvgTotal) / worst.AvgTotal * 100
			fmt.Printf("🏆 Most efficient: %s (%.0f avg tokens, %.0f%% fewer than %s)\n",
				best.Agent, best.AvgTotal, savings, worst.Agent)
		}

		// Find cheapest (lowest cost) separately from most efficient
		cheapest := summaries[0]
		expensive := summaries[len(summaries)-1]
		for _, s := range summaries {
			if s.AvgCost > 0 && (cheapest.AvgCost == 0 || s.AvgCost < cheapest.AvgCost) {
				cheapest = s
			}
			if s.AvgCost > expensive.AvgCost {
				expensive = s
			}
		}
		if cheapest.AvgCost > 0 && expensive.AvgCost > cheapest.AvgCost {
			costSavings := (expensive.AvgCost - cheapest.AvgCost) / expensive.AvgCost * 100
			fmt.Printf("💰 Cheapest: %s (%s avg vs %s for %s, %.0f%% savings)\n",
				cheapest.Agent, fmtCost(cheapest.AvgCost), fmtCost(expensive.AvgCost), expensive.Agent, costSavings)
		}
	}
}
