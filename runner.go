package bench

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Runner executes a single agent on a task against a repo.
type Runner struct {
	WorkDir  string // base directory for cloned repos
	Agent    string // agent name: inber, openclaw, claude-code
	Task     string // path to task markdown file
	RepoURL  string // git clone URL
	Commit   string // git commit to reset to
	BuildCmd string // build command (e.g. "go build ./...")
	TestCmd  string // test command (e.g. "go test ./...")
}

// Run executes the benchmark and returns the result.
func (r *Runner) Run() (*Result, error) {
	result := &Result{
		Agent:     r.Agent,
		Task:      filepath.Base(r.Task),
		Repo:      r.RepoURL,
		Commit:    r.Commit,
		Timestamp: time.Now(),
	}

	// Read task prompt
	taskContent, err := os.ReadFile(r.Task)
	if err != nil {
		return nil, fmt.Errorf("read task: %w", err)
	}

	// Prepare repo clone
	repoDir := filepath.Join(r.WorkDir, r.Agent)
	if err := r.prepareRepo(repoDir); err != nil {
		return nil, fmt.Errorf("prepare repo: %w", err)
	}

	// Run the agent
	start := time.Now()
	agentOutput, err := r.runAgent(repoDir, string(taskContent))
	elapsed := time.Since(start)

	result.Metrics.WallTime = elapsed
	result.Metrics.WallTimeSec = elapsed.Seconds()

	if err != nil {
		result.Error = err.Error()
	}

	// Parse agent-specific metrics from output
	r.parseMetrics(agentOutput, &result.Metrics)

	// Collect git stats
	result.Git = r.collectGitStats(repoDir)

	// Run quality checks
	result.Quality = r.checkQuality(repoDir)

	return result, nil
}

func (r *Runner) prepareRepo(dir string) error {
	// Clean and clone
	os.RemoveAll(dir)
	if err := run("git", "clone", r.RepoURL, dir); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	if r.Commit != "" {
		if err := runIn(dir, "git", "checkout", r.Commit); err != nil {
			return fmt.Errorf("checkout %s: %w", r.Commit, err)
		}
	}
	return nil
}

func (r *Runner) runAgent(repoDir, prompt string) (string, error) {
	switch r.Agent {
	case "inber":
		return r.runInber(repoDir, prompt)
	case "claude-code":
		return r.runClaudeCode(repoDir, prompt)
	case "openclaw":
		return r.runOpenClaw(repoDir, prompt)
	default:
		return "", fmt.Errorf("unknown agent: %s", r.Agent)
	}
}

func (r *Runner) runInber(repoDir, prompt string) (string, error) {
	cmd := exec.Command("inber", "run", "--new", "--detach", prompt)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *Runner) runClaudeCode(repoDir, prompt string) (string, error) {
	cmd := exec.Command("claude", "-p", prompt, "--output-format", "json", "--max-turns", "20")
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *Runner) runOpenClaw(repoDir, prompt string) (string, error) {
	// TODO: spawn via openclaw CLI or API
	return "", fmt.Errorf("openclaw runner not yet implemented")
}

func (r *Runner) parseMetrics(output string, m *Metrics) {
	// Parse inber-style output: "in=63593 out=789 total=64382 tools=9"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "in=") && strings.Contains(line, "out=") {
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "in=") {
					m.InputTokens, _ = strconv.Atoi(strings.TrimPrefix(part, "in="))
				}
				if strings.HasPrefix(part, "out=") {
					m.OutputTokens, _ = strconv.Atoi(strings.TrimPrefix(part, "out="))
				}
				if strings.HasPrefix(part, "total=") {
					m.TotalTokens, _ = strconv.Atoi(strings.TrimPrefix(part, "total="))
				}
				if strings.HasPrefix(part, "tools=") {
					m.ToolCalls, _ = strconv.Atoi(strings.TrimPrefix(part, "tools="))
				}
			}
		}

		if strings.Contains(line, "cost=$") {
			idx := strings.Index(line, "cost=$")
			costStr := line[idx+6:]
			if space := strings.IndexAny(costStr, " \t\n"); space > 0 {
				costStr = costStr[:space]
			}
			m.CostUSD, _ = strconv.ParseFloat(costStr, 64)
		}

		// Count turns from "━━━ Turn N ━━━"
		if strings.Contains(line, "Turn") && strings.Contains(line, "━━━") {
			m.Turns++
		}
	}
}

func (r *Runner) collectGitStats(repoDir string) GitStats {
	stats := GitStats{}

	// git diff --stat
	out, err := output(repoDir, "git", "diff", "--stat", "HEAD")
	if err != nil {
		return stats
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Last line is summary: "N files changed, N insertions(+), N deletions(-)"
		if strings.Contains(line, "files changed") || strings.Contains(line, "file changed") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				stats.FilesChanged, _ = strconv.Atoi(parts[0])
			}
			for i, p := range parts {
				if strings.HasPrefix(p, "insertion") && i > 0 {
					stats.LinesAdded, _ = strconv.Atoi(parts[i-1])
				}
				if strings.HasPrefix(p, "deletion") && i > 0 {
					stats.LinesRemoved, _ = strconv.Atoi(parts[i-1])
				}
			}
		}
	}

	// git diff --name-only
	nameOut, _ := output(repoDir, "git", "diff", "--name-only", "HEAD")
	for _, f := range strings.Split(strings.TrimSpace(nameOut), "\n") {
		f = strings.TrimSpace(f)
		if f != "" {
			stats.ChangedFiles = append(stats.ChangedFiles, f)
		}
	}

	return stats
}

func (r *Runner) checkQuality(repoDir string) QualityCheck {
	q := QualityCheck{}

	buildCmd := r.BuildCmd
	if buildCmd == "" {
		buildCmd = "go build ./..."
	}
	testCmd := r.TestCmd
	if testCmd == "" {
		testCmd = "go test ./..."
	}

	// Build
	q.Builds = runIn(repoDir, "sh", "-c", buildCmd) == nil

	// Test
	testOut, err := output(repoDir, "sh", "-c", testCmd)
	q.TestsPass = err == nil
	q.TestOutput = testOut

	return q
}

// helpers

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runIn(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

func output(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
