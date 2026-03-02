package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	bench "github.com/kayushkin/agent-bench"
	"github.com/spf13/cobra"
)

var agents = []string{"inber", "claude-code", "openclaw"}

func main() {
	root := &cobra.Command{
		Use:   "agent-bench",
		Short: "Benchmark AI coding agents on identical tasks",
	}

	// Run command
	var (
		agent    string
		all      bool
		task     string
		repo     string
		commit   string
		buildCmd string
		testCmd  string
		outDir   string
	)

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a benchmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			if task == "" {
				return fmt.Errorf("--task is required")
			}
			if repo == "" {
				return fmt.Errorf("--repo is required")
			}

			if outDir == "" {
				outDir = filepath.Join("results", time.Now().Format("2006-01-02"))
			}
			workDir := filepath.Join("work", time.Now().Format("2006-01-02_150405"))

			agentsToRun := []string{agent}
			if all {
				agentsToRun = agents
			}

			for _, a := range agentsToRun {
				fmt.Printf("\n═══ Running %s ═══\n\n", a)

				r := &bench.Runner{
					WorkDir:  workDir,
					Agent:    a,
					Task:     task,
					RepoURL:  repo,
					Commit:   commit,
					BuildCmd: buildCmd,
					TestCmd:  testCmd,
				}

				result, err := r.Run()
				if err != nil {
					fmt.Fprintf(os.Stderr, "error running %s: %v\n", a, err)
					continue
				}

				if err := bench.SaveResult(outDir, result); err != nil {
					fmt.Fprintf(os.Stderr, "error saving result: %v\n", err)
				}

				fmt.Printf("  tokens: in=%d out=%d total=%d\n", result.Metrics.InputTokens, result.Metrics.OutputTokens, result.Metrics.TotalTokens)
				fmt.Printf("  cost: $%.4f\n", result.Metrics.CostUSD)
				fmt.Printf("  time: %.1fs\n", result.Metrics.WallTimeSec)
				fmt.Printf("  git: %d files, +%d -%d\n", result.Git.FilesChanged, result.Git.LinesAdded, result.Git.LinesRemoved)
				fmt.Printf("  build: %v  tests: %v\n", result.Quality.Builds, result.Quality.TestsPass)
			}

			return nil
		},
	}

	runCmd.Flags().StringVarP(&agent, "agent", "a", "inber", "Agent to benchmark (inber, claude-code, openclaw)")
	runCmd.Flags().BoolVar(&all, "all", false, "Run all agents")
	runCmd.Flags().StringVarP(&task, "task", "t", "", "Path to task markdown file")
	runCmd.Flags().StringVarP(&repo, "repo", "r", "", "Git repo URL to clone")
	runCmd.Flags().StringVarP(&commit, "commit", "c", "", "Git commit to reset to")
	runCmd.Flags().StringVar(&buildCmd, "build-cmd", "", "Build command (default: go build ./...)")
	runCmd.Flags().StringVar(&testCmd, "test-cmd", "", "Test command (default: go test ./...)")
	runCmd.Flags().StringVarP(&outDir, "out", "o", "", "Output directory for results")

	// Compare command
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
