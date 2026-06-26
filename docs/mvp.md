# Mini Agent Orchestrator v0 — MVP

## Goal

Create a Go CLI tool that transforms a raw project idea into a fixed set of markdown artifacts.

## Input

The tool accepts either:

- a short idea via `--idea`
- a markdown file via `--input`

## Output

The tool creates a project folder with:

- `idea.md`
- `spec.md`
- `tasks.md`
- `review-checklist.md`
- `metadata.json`

## Out of scope for v0

- LLM API calls
- GitHub integration
- Web UI
- configurable workflows
- plugins
- memory
- multi-agent orchestration

## Success criteria

The command works:

```bash
mao run --idea "Build a personal book library" --out ./artifacts/demo
```

Running this command creates the `./artifacts/demo` folder containing all five artifacts (`idea.md`, `spec.md`, `tasks.md`, `review-checklist.md`, and `metadata.json`).