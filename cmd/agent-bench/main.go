package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	bench "github.com/kayushkin/agent-bench"
	"github.com/spf13/cobra"
)

var defaultAgents = []string{"inber", "openclaw"}

func main() {
	root := &cobra.Command{
		Use:   "agent-bench",
		Short: "Benchmark AI coding agents on identical tasks",
	}

	var (
		agent    string
		all      bool
		task     string
		repo     string
		repoDir  string
		commit   string
		buildCmd string
		testCmd  string
		outDir   string
		maxTurns int
		model    string
		trials   int
	)

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a benchmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			if task == "" {
				return fmt.Errorf("--task is required")
			}
			if repo == "" && repoDir == "" {
				return fmt.Errorf("--repo or --dir is required")
			}

			timestamp := time.Now().Format("2006-01-02_150405")
			if outDir == "" {
				outDir = filepath.Join("results", timestamp)
			}

			agentsToRun := []string{agent}
			if all {
				agentsToRun = defaultAgents
			}
			if trials < 1 {
				trials = 1
			}

			// Auto-detect per-task testdata: testdata/{task-name}/ takes priority over --dir
			effectiveRepoDir := repoDir
			if effectiveRepoDir != "" {
				taskBase := filepath.Base(task)
				taskName := taskBase[:len(taskBase)-len(filepath.Ext(taskBase))]
				perTaskDir := filepath.Join("testdata", taskName)
				if info, err := os.Stat(perTaskDir); err == nil && info.IsDir() {
					fmt.Printf("Using per-task testdata: %s\n", perTaskDir)
					effectiveRepoDir = perTaskDir
				}
			}

			for trial := 1; trial <= trials; trial++ {
				for _, a := range agentsToRun {
					trialLabel := ""
					if trials > 1 {
						trialLabel = fmt.Sprintf(" (trial %d/%d)", trial, trials)
					}
					fmt.Printf("\n═══ %s%s ═══\n\n", a, trialLabel)

					workDir := filepath.Join("work", timestamp, fmt.Sprintf("trial%d", trial))
					os.MkdirAll(workDir, 0755)

					r := &bench.Runner{
						WorkDir:  workDir,
						Agent:    a,
						Task:     task,
						RepoURL:  repo,
						RepoDir:  effectiveRepoDir,
						Commit:   commit,
						BuildCmd: buildCmd,
						TestCmd:  testCmd,
						MaxTurns: maxTurns,
						Model:    model,
						Trial:    trial,
					}

					result, err := r.Run()
					if err != nil {
						fmt.Fprintf(os.Stderr, "error running %s: %v\n", a, err)
						continue
					}

					if err := bench.SaveResult(outDir, result); err != nil {
						fmt.Fprintf(os.Stderr, "error saving result: %v\n", err)
					}

					m := result.Metrics
					q := result.Quality
					fmt.Printf("  model:  %s\n", m.Model)
					fmt.Printf("  tokens: in=%d out=%d total=%d", m.InputTokens, m.OutputTokens, m.TotalTokens)
					if m.CacheReadTokens > 0 {
						fmt.Printf(" cache_read=%d", m.CacheReadTokens)
					}
					if m.CacheCreationTokens > 0 {
						fmt.Printf(" cache_write=%d", m.CacheCreationTokens)
					}
					fmt.Println()
					fmt.Printf("  cost:   %s\n", fmtCostCLI(m.CostUSD))
					fmt.Printf("  turns:  %d  tools: %d  time: %.1fs\n", m.Turns, m.ToolCalls, m.WallTimeSec)
					fmt.Printf("  git:    %d files, +%d -%d\n", result.Git.FilesChanged, result.Git.LinesAdded, result.Git.LinesRemoved)
					fmt.Printf("  build:  %v  tests: %v\n", q.Builds, q.TestsPass)
					if result.Error != "" {
						fmt.Printf("  error:  %s\n", result.Error)
					}
					fmt.Println()
				}
			}

			// Print summary if multiple trials
			if trials > 1 {
				report, err := bench.LoadResults(outDir)
				if err == nil {
					report.PrintComparison()
				}
			}

			return nil
		},
	}

	runCmd.Flags().StringVarP(&agent, "agent", "a", "inber", "Agent to benchmark")
	runCmd.Flags().BoolVar(&all, "all", false, "Run all agents")
	runCmd.Flags().StringVarP(&task, "task", "t", "", "Path to task markdown file")
	runCmd.Flags().StringVarP(&repo, "repo", "r", "", "Git repo URL to clone")
	runCmd.Flags().StringVarP(&repoDir, "dir", "d", "", "Local repo directory to copy")
	runCmd.Flags().StringVarP(&commit, "commit", "c", "", "Git commit to reset to")
	runCmd.Flags().StringVar(&buildCmd, "build-cmd", "", "Build command (default: go build ./...)")
	runCmd.Flags().StringVar(&testCmd, "test-cmd", "", "Test command (default: go test ./...)")
	runCmd.Flags().StringVarP(&outDir, "out", "o", "", "Output directory for results")
	runCmd.Flags().IntVar(&maxTurns, "max-turns", 15, "Max agent turns")
	runCmd.Flags().StringVarP(&model, "model", "m", "claude-sonnet-4-5-20250929", "Model for both agents")
	runCmd.Flags().IntVarP(&trials, "trials", "n", 1, "Number of trials per agent")

	compareCmd := &cobra.Command{
		Use:   "compare [results-dir]",
		Short: "Compare benchmark results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := bench.LoadResults(args[0])
			if err != nil {
				return err
			}
			report.PrintComparison()
			return nil
		},
	}

	root.AddCommand(runCmd, compareCmd)
	root.Execute()
}

func fmtCostCLI(c float64) string {
	if c == 0 {
		return "$0"
	}
	if c < 0.001 {
		return fmt.Sprintf("$%.5f", c)
	}
	return fmt.Sprintf("$%.4f", c)
}
