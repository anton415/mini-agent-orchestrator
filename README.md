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

## Example

A committed sample input is available at `examples/book-library.md`.
The corresponding expected artifact set is checked in at `examples/expected-output/book-library/`, outside the ignored `/artifacts/` directory.
To regenerate a byte-for-byte comparable fixture, pass the fixed timestamp used by the committed metadata:

```bash
mao run --input ./examples/book-library.md --out ./examples/expected-output --name book-library --created-at 2026-06-25T11:23:10Z --force
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

## v0.2 Scope

v0.2 is scoped to optional prompt-template generation for the existing deterministic workflow.
See `docs/v0.2.md` for the planned prompt artifact set and explicit non-goals.
