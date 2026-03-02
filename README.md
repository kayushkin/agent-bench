# agent-bench

Benchmark harness for comparing AI coding agents on identical tasks.

## How it works

1. Clone a target repo to separate dirs (one per agent)
2. Feed each agent the same task prompt
3. Collect metrics: tokens, cost, time, code quality
4. Generate comparison report

## Agents

- **inber** — `inber run` with session JSONL parsing
- **openclaw** — sub-agent spawn via CLI
- **claude-code** — `claude -p` with JSON output

## Usage

```bash
# Run a single agent on a task
agent-bench run --agent inber --task tasks/01-add-endpoint.md --repo github.com/kayushkin/inber --commit abc123

# Run all agents on a task
agent-bench run --all --task tasks/01-add-endpoint.md --repo github.com/kayushkin/inber --commit abc123

# Compare results
agent-bench compare results/2026-03-02/
```

## Task Format

Tasks are markdown files in `tasks/`:

```markdown
# Add health endpoint

Add a GET /health endpoint that returns:
{"status": "ok", "version": "1.0.0", "uptime_seconds": <seconds since start>}

## Acceptance Criteria
- Returns 200 with JSON
- Includes uptime in seconds
- Has a test
```

## Metrics

| Metric | Source |
|--------|--------|
| Input tokens | Agent logs |
| Output tokens | Agent logs |
| Cost (USD) | Agent logs |
| Turns / tool calls | Agent logs |
| Wall time | Harness timer |
| Files changed | git diff |
| Lines +/- | git diff |
| Build pass | go build / npm build |
| Tests pass | go test / npm test |
| Scope creep score | Files changed outside task scope |
