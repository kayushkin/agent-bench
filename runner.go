package bench

import (
	"bufio"
	"encoding/json"
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
	Agent    string // agent name: inber, openclaw
	Task     string // path to task markdown file
	RepoDir  string // local repo directory (used directly, no cloning)
	RepoURL  string // git clone URL (alternative to RepoDir)
	Commit   string // git commit to reset to
	BuildCmd string // build command (e.g. "go build ./...")
	TestCmd  string // test command (e.g. "go test ./...")
	MaxTurns int    // max agent turns (default 15)
	Model    string // model to use (e.g. "claude-sonnet-4-20250514")
	Trial    int    // trial number (for multi-trial runs)
}

// Run executes the benchmark and returns the result.
func (r *Runner) Run() (*Result, error) {
	result := &Result{
		Agent:     r.Agent,
		Task:      filepath.Base(r.Task),
		Trial:     r.Trial,
		Timestamp: time.Now(),
	}

	// Read task prompt
	taskContent, err := os.ReadFile(r.Task)
	if err != nil {
		return nil, fmt.Errorf("read task: %w", err)
	}

	// Prepare repo: either use local dir or clone
	repoDir := r.RepoDir
	if repoDir == "" {
		repoDir = filepath.Join(r.WorkDir, r.Agent)
		result.Repo = r.RepoURL
		result.Commit = r.Commit
		if err := r.prepareRepo(repoDir); err != nil {
			return nil, fmt.Errorf("prepare repo: %w", err)
		}
	} else {
		// Copy the local repo to a working directory per agent
		workCopy := filepath.Join(r.WorkDir, r.Agent)
		if err := copyDir(repoDir, workCopy); err != nil {
			return nil, fmt.Errorf("copy repo: %w", err)
		}
		repoDir = workCopy
		result.Repo = repoDir
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

	// Set model if known from runner config but not from output
	if result.Metrics.Model == "" && r.Model != "" {
		result.Metrics.Model = r.Model
	}

	// Calculate cost
	result.Metrics.CalculateCost()

	return result, nil
}

func (r *Runner) prepareRepo(dir string) error {
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
	case "openclaw":
		return r.runOpenClaw(repoDir, prompt)
	default:
		return "", fmt.Errorf("unknown agent: %s", r.Agent)
	}
}

func (r *Runner) runInber(repoDir, prompt string) (string, error) {
	maxTurns := r.MaxTurns
	if maxTurns == 0 {
		maxTurns = 15
	}

	args := []string{"run", "--new", "--max-turns", strconv.Itoa(maxTurns)}
	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}
	args = append(args, prompt)

	cmd := exec.Command("inber", args...)
	cmd.Dir = repoDir

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *Runner) runOpenClaw(repoDir, prompt string) (string, error) {
	absRepoDir, _ := filepath.Abs(repoDir)

	// Ensure agent-bench has auth
	r.pinOpenClawModel(r.Model)

	sessionID := fmt.Sprintf("bench-%d", time.Now().UnixMilli())

	// Prefix prompt with working directory context
	fullPrompt := fmt.Sprintf("You are working in the directory: %s\n\nIMPORTANT: Use this exact path for all file operations. Do not read AGENTS.md, SOUL.md, USER.md, or any workspace files. Focus only on the task.\n\n%s", absRepoDir, prompt)

	cmd := exec.Command("openclaw", "agent",
		"--local",
		"--json",
		"--agent", "agent-bench",
		"--session-id", sessionID,
		"--message", fullPrompt,
		"--timeout", "300",
	)
	cmd.Dir = absRepoDir

	// Capture stdout and stderr separately — openclaw writes
	// diagnostic lines to stderr and JSON result to stdout
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	// Save raw output for debugging
	debugDir := filepath.Join(r.WorkDir, "debug")
	os.MkdirAll(debugDir, 0755)
	os.WriteFile(filepath.Join(debugDir, "openclaw-stdout.txt"), []byte(stdout.String()), 0644)
	os.WriteFile(filepath.Join(debugDir, "openclaw-stderr.txt"), []byte(stderr.String()), 0644)

	// Return stdout only — JSON output is on stdout, stderr has diagnostic noise
	// that causes json.Unmarshal to fail with "extra data"
	return stdout.String(), err
}

// pinOpenClawModel copies main agent's auth to agent-bench.
// Model pinning not yet implemented — openclaw uses gateway defaults.
func (r *Runner) pinOpenClawModel(model string) {
	home := os.Getenv("HOME")
	ocDir := filepath.Join(home, ".openclaw")

	// Copy main agent's auth-profiles to agent-bench
	mainAuth := filepath.Join(ocDir, "agents", "main", "agent", "auth-profiles.json")
	benchAuth := filepath.Join(ocDir, "agents", "agent-bench", "agent", "auth-profiles.json")
	if data, err := os.ReadFile(mainAuth); err == nil {
		os.WriteFile(benchAuth, data, 0644)
	}
}

func (r *Runner) parseMetrics(output string, m *Metrics) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse INBER_META:{json} from stderr (inber)
		if strings.HasPrefix(line, "INBER_META:") {
			metaJSON := strings.TrimPrefix(line, "INBER_META:")
			var meta struct {
				InputTokens  int     `json:"input_tokens"`
				OutputTokens int     `json:"output_tokens"`
				TotalTokens  int     `json:"total_tokens"`
				ToolCalls    int     `json:"tool_calls"`
				Turns        int     `json:"turns"`
				CostUSD      float64 `json:"cost_usd"`
			}
			if err := json.Unmarshal([]byte(metaJSON), &meta); err == nil {
				m.InputTokens = meta.InputTokens
				m.OutputTokens = meta.OutputTokens
				m.TotalTokens = meta.TotalTokens
				m.ToolCalls = meta.ToolCalls
				m.Turns = meta.Turns
				m.CostUSD = meta.CostUSD
			}
			continue
		}

		// Fallback: parse "in=X out=Y total=Z tools=N" format
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

		// Count turns from "━━━ Turn N ━━━"
		if strings.Contains(line, "Turn") && strings.Contains(line, "━━━") {
			m.Turns++
		}
	}

	// Try parsing openclaw JSON (--json output).
	// The output may be pretty-printed, so find the first '{' and parse from there.
	if idx := strings.Index(output, "{\n"); idx >= 0 && strings.Contains(output, `"payloads"`) {
		jsonStr := output[idx:]
		var ocResult struct {
			Meta struct {
				DurationMs int `json:"durationMs"`
				AgentMeta  struct {
					Model string `json:"model"`
					Usage struct {
						Input     int `json:"input"`
						Output    int `json:"output"`
						CacheRead int `json:"cacheRead"`
						Total     int `json:"total"`
					} `json:"usage"`
				} `json:"agentMeta"`
			} `json:"meta"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &ocResult); err == nil && (ocResult.Meta.AgentMeta.Usage.Input > 0 || ocResult.Meta.AgentMeta.Usage.Output > 0) {
			u := ocResult.Meta.AgentMeta.Usage
			m.InputTokens = u.Input
			m.OutputTokens = u.Output
			m.CacheReadTokens = u.CacheRead
			// OpenClaw's "total" includes cache read; use input+output for fair comparison
			m.TotalTokens = u.Input + u.Output
			m.Model = ocResult.Meta.AgentMeta.Model
		}
	}

	// Set total if not parsed
	if m.TotalTokens == 0 && (m.InputTokens > 0 || m.OutputTokens > 0) {
		m.TotalTokens = m.InputTokens + m.OutputTokens
	}
}

func (r *Runner) collectGitStats(repoDir string) GitStats {
	stats := GitStats{}

	// Find the original commit (first commit in repo) to diff against.
	// This captures both committed and uncommitted changes the agent made.
	baseCommit := "HEAD"
	firstCommit, err := output(repoDir, "git", "rev-list", "--max-parents=0", "HEAD")
	if err == nil {
		fc := strings.TrimSpace(firstCommit)
		if fc != "" {
			baseCommit = fc
		}
	}

	// Include both staged+unstaged changes: diff from base to working tree
	out, err := output(repoDir, "git", "diff", "--stat", baseCommit)
	if err != nil {
		return stats
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
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

	nameOut, _ := output(repoDir, "git", "diff", "--name-only", baseCommit)
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

	q.Builds = runIn(repoDir, "sh", "-c", buildCmd) == nil

	// Clear test cache so we get a real run
	runIn(repoDir, "go", "clean", "-testcache")
	testOut, err := output(repoDir, "sh", "-c", testCmd)
	q.TestsPass = err == nil
	q.TestOutput = testOut

	return q
}

// copyDir copies src directory to dst and initializes a fresh git repo.
func copyDir(src, dst string) error {
	os.RemoveAll(dst)
	// Copy without preserving git metadata
	if err := runCmd("cp", "-r", src, dst); err != nil {
		return err
	}
	// Remove any inherited .git from parent
	os.RemoveAll(filepath.Join(dst, ".git"))
	// Initialize fresh repo
	if err := runIn(dst, "git", "init"); err != nil {
		return err
	}
	if err := runIn(dst, "git", "add", "-A"); err != nil {
		return err
	}
	return runIn(dst, "git", "commit", "-m", "initial")
}

// helpers

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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

// outputLines runs a command and returns stdout lines via a scanner callback.
func outputLines(dir, name string, args ...string) *bufio.Scanner {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	pipe, _ := cmd.StdoutPipe()
	cmd.Start()
	return bufio.NewScanner(pipe)
}
