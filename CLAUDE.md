# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Rule 1 — Think Before Coding
State assumptions explicitly. If uncertain, ask rather than guess.
Present multiple interpretations when ambiguity exists.
Push back when a simpler approach exists.
Stop when confused. Name what's unclear.l

## Rule 2 — Simplicity First
Minimum code that solves the problem. Nothing speculative.
No features beyond what was asked. No abstractions for single-use code.
Test: would a senior engineer say this is overcomplicated? If yes, simplify.

## Rule 3 — Surgical Changes
Touch only what you must. Clean up only your own mess.
Don't "improve" adjacent code, comments, or formatting.
Don't refactor what isn't broken. Match existing style.

## Rule 4 — Goal-Driven Execution
Define success criteria. Loop until verified.
Don't follow steps. Define success and iterate.
Strong success criteria let you loop independently.

## Rule 5 — Use the model only for judgment calls
Use me for: classification, drafting, summarization, extraction.
Do NOT use me for: routing, retries, deterministic transforms.
If code can answer, code answers.

## Rule 6 — Token budgets are not advisory
Per-task: 4,000 tokens. Per-session: 30,000 tokens.
If approaching budget, summarize and start fresh.
Surface the breach. Do not silently overrun.

## Rule 7 — Surface conflicts, don't average them
If two patterns contradict, pick one (more recent / more tested).
Explain why. Flag the other for cleanup.
Don't blend conflicting patterns.

## Rule 8 — Read before you write
Before adding code, read exports, immediate callers, shared utilities.
"Looks orthogonal" is dangerous. If unsure why code is structured a way, ask.

## Rule 9 — Tests verify intent, not just behavior
Tests must encode WHY behavior matters, not just WHAT it does.
A test that can't fail when business logic changes is wrong.

## Rule 10 — Checkpoint after every significant step
Summarize what was done, what's verified, what's left.
Don't continue from a state you can't describe back.
If you lose track, stop and restate.

## Rule 11 — Match the codebase's conventions, even if you disagree
Conformance > taste inside the codebase.
If you genuinely think a convention is harmful, surface it. Don't fork silently.

## Rule 12 — Fail loud
"Completed" is wrong if anything was skipped silently.
"Tests pass" is wrong if any were skipped.
Default to surfacing uncertainty, not hiding it.


## Repository status

This repository is a bootstrap-stage Go project. As of this writing, the only committed files are `go.mod` and `go.sum` — there is no source code, no `main` package, no tests, no README, and no build/lint configuration. The module name is `github.com/shahensargsyan/my-new-go-api` and Go version is `1.25.1`.

Because there is no code, do not assume any directory layout or package structure. When asked to add functionality, ask the user where it should live unless they specify.

## Intended stack (inferred from go.mod)

All dependencies in `go.mod` are currently `// indirect`, which means nothing in this module imports them directly yet — they were pulled in only as transitive deps or pre-staged for upcoming work. Treat the list as the intended stack, not as a contract:

- **HTTP framework:** `github.com/gin-gonic/gin`
- **Relational DB:** `gorm.io/gorm` + `gorm.io/driver/mysql` (MySQL via GORM)
- **Document DB:** `go.mongodb.org/mongo-driver/v2` (MongoDB)
- **Validation:** `github.com/go-playground/validator/v10`
- **QUIC/HTTP3:** `github.com/quic-go/quic-go`

When you write the first code that uses any of these, promote them out of the `indirect` block by importing them and running `go mod tidy`.

## Common commands

Standard Go tooling applies — there is no Makefile or task runner.

- Build: `go build ./...`
- Run module tidy: `go mod tidy`
- Vet: `go vet ./...`
- Test all: `go test ./...`
- Test single package: `go test ./path/to/pkg`
- Test single function: `go test ./path/to/pkg -run TestName`
- Format: `gofmt -w .` (or `go fmt ./...`)
