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
	path := filepath.Join(dir, r.Agent+".json")
	return os.WriteFile(path, data, 0644)
}

// PrintComparison outputs a side-by-side comparison.
func (r *Report) PrintComparison() {
	if len(r.Results) == 0 {
		fmt.Println("No results to compare.")
		return
	}

	// Sort by total tokens (ascending = more efficient first)
	sort.Slice(r.Results, func(i, j int) bool {
		return r.Results[i].Metrics.TotalTokens < r.Results[j].Metrics.TotalTokens
	})

	task := r.Results[0].Task

	fmt.Printf("\n═══ Agent Benchmark: %s ═══\n\n", task)
	fmt.Printf("%-14s %10s %10s %10s %8s %6s %6s %6s %5s %5s\n",
		"Agent", "In Tokens", "Out Tokens", "Total", "Cost", "Turns", "Tools", "Time", "Build", "Test")
	fmt.Println(strings.Repeat("─", 95))

	for _, res := range r.Results {
		m := res.Metrics
		q := res.Quality
		buildStr := "✓"
		if !q.Builds {
			buildStr = "✗"
		}
		testStr := "✓"
		if !q.TestsPass {
			testStr = "✗"
		}

		fmt.Printf("%-14s %10d %10d %10d %7s %6d %6d %5.0fs %5s %5s\n",
			res.Agent,
			m.InputTokens,
			m.OutputTokens,
			m.TotalTokens,
			fmt.Sprintf("$%.2f", m.CostUSD),
			m.Turns,
			m.ToolCalls,
			m.WallTimeSec,
			buildStr,
			testStr,
		)
	}

	fmt.Println()

	// Git diff comparison
	fmt.Printf("%-14s %8s %8s %8s %s\n", "Agent", "Files", "Added", "Removed", "Scope Creep")
	fmt.Println(strings.Repeat("─", 60))
	for _, res := range r.Results {
		g := res.Git
		fmt.Printf("%-14s %8d %8d %8d %d files\n",
			res.Agent,
			g.FilesChanged,
			g.LinesAdded,
			g.LinesRemoved,
			res.Quality.ScopeCreep,
		)
	}

	// Winner summary
	fmt.Println()
	if len(r.Results) > 1 {
		best := r.Results[0]
		worst := r.Results[len(r.Results)-1]
		savings := float64(worst.Metrics.TotalTokens-best.Metrics.TotalTokens) / float64(worst.Metrics.TotalTokens) * 100
		fmt.Printf("🏆 Most efficient: %s (%d tokens, %.0f%% fewer than %s)\n",
			best.Agent, best.Metrics.TotalTokens, savings, worst.Agent)
	}
}
