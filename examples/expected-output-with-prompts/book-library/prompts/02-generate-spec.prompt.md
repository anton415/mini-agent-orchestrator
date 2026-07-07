# Prompt: Generate Specification

## Project

book-library

## Raw idea

Build a personal book library that lets a reader catalog books they own, track reading status, and record simple notes or ratings. Start with a local single-user workflow and defer social features.

## Task

Create the `spec.md` artifact for the fixed workflow. If a normalized `idea.md` exists, use it as the source of truth together with the raw idea. Describe the smallest useful first version in concrete, reviewable terms.

## Expected output

Return Markdown only, using this structure:

```markdown
# Specification

## Goal

- Problem being solved:
- Value delivered by the first usable version:
- One-sentence success statement:

## User

- Primary user:
- User context or environment:
- Current workaround or pain:
- Reason this matters now:

## Use cases

- As a [user], I want [action], so I can [outcome].
- As a [user], I want [action], so I can [outcome].
- As a [user], I want [action], so I can [outcome].

## Functional requirements

- The system should ...
- The system should ...
- The system should ...

## Non-functional requirements

- Reliability:
- Performance:
- Security and privacy:
- Accessibility:
- Maintainability:

## Constraints

- Platforms or runtimes:
- Data sources or formats:
- Integration limits:
- Budget, timeline, or staffing:

## Out of scope

- Deferred:
- Deferred:
- Deferred:

## Success criteria

- A user can ...
- The project handles ...
- The project avoids ...

## Open questions

- Question:
- Question:
- Question:
```

## Constraints

- Do not add out-of-scope features.
- Do not use network access, API calls, or API keys.
- Keep requirements observable and testable.
- Separate confirmed constraints from assumptions.
- Defer tempting features that do not support the first useful workflow.
