# Project Instructions

Defensive code and backward compatibility are not required.
Implement only the minimum code and tests needed to achieve the MVP.

## Project Goal

`zenn-topics` is a Go CLI that fetches `topics*.xml.gz` files from the Zenn sitemap, parses the XML, and returns a list of topics.

MVP scope:

- `zenn-topics <query>` searches topics and prints matches.

## Phase Plan

- Phase0 (mostly user-owned): development environment setup (`mise` tasks, hooks, `golangci-lint`, `goreleaser`, CI/release workflows, `bump.sh`).
- Phase0 (agent-owned): maintain `AGENTS.md` and `CLAUDE.md` in English, and create/update `TODO.md` for Phase1.
- Phase1: implement the MVP command `zenn-topics <query>`.

## Required Workflow

Follow this workflow for each task unless the user explicitly changes it:

1. Wait for the user to assign the next task.
2. Read `AGENTS.md` and `TODO.md` before starting work.
3. Refine the task, define DoD (Definition of Done), detail the relevant part of `TODO.md`, and ask follow-up questions only if needed.
4. Consider high-priority product improvement ideas related to the current task while refining the task/TODOs.
5. Implement the task.
6. Verify until `mise run` checks, DoD, and pre-commit checks pass, then create a commit.
7. Ask implementation questions (if any) and request user review.
8. Address user review feedback in a loop until finished.
9. After user review is completed, consider high-priority product improvement ideas before moving on.
10. Push only after pre-push checks pass.

## Engineering Priorities

- Build the minimum implementation that solves the current requirement.
- Backward compatibility is not a priority during MVP development.
- Prefer simple, explicit code over premature abstraction.

## Testing Policy

Use classical, pure, state-based tests as the default approach.

- Test behavior of public APIs.
- Prefer deterministic fixtures and pure functions.
- Isolate network I/O from parsing/search logic so core behavior is easy to test.
- Avoid tests that depend on live external services unless explicitly requested.
- Defensive tests and tests that assume backward compatibility are not required.
- When adding or keeping tests, optimize for maintenance value over branch count or code coverage.

## Improvement Triggers

- If the user points out issues in how work was executed, improve `AGENTS.md` as needed.
- At task start (when detailing task-specific items in `TODO.md`), consider product improvement ideas related to that task.
- After user review for a task is completed, consider product improvement ideas again based on the review outcome.
- Skip low-priority improvements unless they are important enough to justify the interruption.
- Avoid duplicating rules or guarantees that are already enforced deterministically by linters/CI.
