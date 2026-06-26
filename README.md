# Mini Agent Orchestrator

[![Go](https://github.com/anton415/mini-agent-orchestrator/actions/workflows/go.yml/badge.svg)](https://github.com/anton415/mini-agent-orchestrator/actions/workflows/go.yml)

Mini Agent Orchestrator is a small Go CLI tool that turns a raw project idea into a fixed set of development artifacts.

## Status

v0 — deterministic CLI workflow runner.

## Usage

```bash
mao run --idea "Build a personal book library" --out ./artifacts --name book-library
```

Or:

```bash
mao run --input ./examples/book-library.md --out ./artifacts --name book-library
```

## Output
```
artifacts/book-library/
  idea.md
  spec.md
  tasks.md
  review-checklist.md
  metadata.json
```

## v0 Scope

Included:
* idea input
* markdown file input
* fixed workflow
* markdown artifact generation
* dry run
* overwrite protection

Not included:
* LLM API
* GitHub integration
* Web UI
* configurable workflows