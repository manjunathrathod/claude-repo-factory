---
description: Implement a feature end to end - plan, build, test, verify against the definition of done
argument-hint: <what to build>
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
---

Implement this feature in `claude-repo-factory`: **$ARGUMENTS**

Follow every rule in `CLAUDE.md`. Work through the phases below in order and
do not skip ahead.

## 1. Orient

- Read the packages the change touches and the ones they depend on.
- State in two or three sentences what you understood the feature to be and
  which packages will change.
- If the request is ambiguous in a way that changes the outcome, ask now.
  Otherwise pick the obvious default, say which, and continue.

## 2. Check the architecture first

Before writing code, answer explicitly:

- Does this add language-specific knowledge? If so it belongs in
  `internal/lang`, not in `internal/cli`, `internal/spec` or `internal/render`.
- Does it require a change to `plugin.Language`? If so, stop and propose an
  ADR in `docs/adr/` before implementing.
- Does it introduce a new dependency, package-level state, or a network call?
  If so, raise it before writing the code.

## 3. Plan

List the files you will create or change and, for each, what it will do. Keep
the plan short — a list, not an essay.

## 4. Build

- Smallest change that fully solves the problem.
- Doc comments on every new exported identifier.
- Errors wrapped with context, never discarded.
- Terminal, filesystem and subprocess access goes behind an interface.
- No refactor of untouched code riding along.

## 5. Test

- Table-driven tests in the external `_test` package.
- Cover the happy path, every error branch, and the boundaries.
- If the feature has a CLI surface, assert on flag resolution, on prompts
  being skipped when a flag was supplied, and on the error message a user
  gets for invalid input.

## 6. Verify

Run all of these and paste the real output:

```
scripts/check.sh
```

That is the validation flow in its canonical order — gofmt, `go vet ./...`,
`go test ./...`, `golangci-lint run`, `go build ./...` — and it stops at the
first failure. Paste its real output. To run a step on its own while
iterating:

```
gofmt -l .
go vet ./...
go test ./...
golangci-lint run
go build ./...
go test -race ./...   # extra gate; unavailable on a 32-bit toolchain
```

## 7. Report

- What you built and where.
- The definition-of-done checklist from `CLAUDE.md` section 12, each item
  marked done, not done or not applicable.
- Anything you deliberately left out, and why.

Do not commit unless the user asks.
