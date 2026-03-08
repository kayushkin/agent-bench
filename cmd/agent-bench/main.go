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
			workDir := filepath.Join("work", timestamp)
			os.MkdirAll(workDir, 0755)

			agentsToRun := []string{agent}
			if all {
				agentsToRun = defaultAgents
			}

			for _, a := range agentsToRun {
				fmt.Printf("\n═══ Running: %s ═══\n\n", a)

				r := &bench.Runner{
					WorkDir:  workDir,
					Agent:    a,
					Task:     task,
					RepoURL:  repo,
					RepoDir:  repoDir,
					Commit:   commit,
					BuildCmd: buildCmd,
					TestCmd:  testCmd,
					MaxTurns: maxTurns,
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
				fmt.Printf("  tokens: in=%d out=%d total=%d\n", m.InputTokens, m.OutputTokens, m.TotalTokens)
				if m.CostUSD > 0 {
					fmt.Printf("  cost:   $%.4f\n", m.CostUSD)
				}
				fmt.Printf("  turns:  %d  tools: %d\n", m.Turns, m.ToolCalls)
				fmt.Printf("  time:   %.1fs\n", m.WallTimeSec)
				fmt.Printf("  git:    %d files, +%d -%d\n", result.Git.FilesChanged, result.Git.LinesAdded, result.Git.LinesRemoved)
				fmt.Printf("  build:  %v  tests: %v\n", q.Builds, q.TestsPass)
				if result.Error != "" {
					fmt.Printf("  error:  %s\n", result.Error)
				}
				fmt.Println()
			}

			return nil
		},
	}

	runCmd.Flags().StringVarP(&agent, "agent", "a", "inber", "Agent to benchmark (inber, openclaw)")
	runCmd.Flags().BoolVar(&all, "all", false, "Run all agents")
	runCmd.Flags().StringVarP(&task, "task", "t", "", "Path to task markdown file")
	runCmd.Flags().StringVarP(&repo, "repo", "r", "", "Git repo URL to clone")
	runCmd.Flags().StringVarP(&repoDir, "dir", "d", "", "Local repo directory to copy")
	runCmd.Flags().StringVarP(&commit, "commit", "c", "", "Git commit to reset to")
	runCmd.Flags().StringVar(&buildCmd, "build-cmd", "", "Build command (default: go build ./...)")
	runCmd.Flags().StringVar(&testCmd, "test-cmd", "", "Test command (default: go test ./...)")
	runCmd.Flags().StringVarP(&outDir, "out", "o", "", "Output directory for results")
	runCmd.Flags().IntVar(&maxTurns, "max-turns", 15, "Max agent turns")

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
