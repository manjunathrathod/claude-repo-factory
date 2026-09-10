---
description: Add a new language or framework plugin without touching core generator logic
argument-hint: <language or framework, e.g. rust, terraform, nextjs>
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
---

Add a language plugin for **$ARGUMENTS** to `claude-repo-factory`.

This workflow exists to prove the architecture holds: adding a language
touches exactly one package. If you find yourself editing anything outside
`internal/lang`, stop and report why.

## 1. Read the pattern

- `internal/lang/lang.go` — the `Definition` and `OptionSpec` shapes.
- `internal/lang/golang.go` and `internal/lang/python.go` — reference
  implementations, one with a single required option and one with a derived
  default.
- `internal/lang/planned.go` — check whether this language is already listed
  as planned.

## 2. Write the plugin

Create `internal/lang/<language>.go` with:

- exported option-key constants the plugin owns;
- a `*Definition` whose `Meta` has an id, display name, summary, aliases that
  collide with nothing else, `plugin.StatusStable`, at least two project
  types, and a `DefaultProjectType` that is one of them;
- `RequiredOptions` covering every answer generation cannot proceed without,
  each with a prompt, help text and a `Default` derived from the Spec;
- `Instruct`, returning `Toolchain`, `Standards`, `Testing`, `Security`,
  `Architecture` and `Commands`;
- an `init` function calling `register(...)`.

If the language was a placeholder, remove its entry from `planned.go`.

## 3. Meet the instruction bar

The instruction text ships into other people's repositories. Be specific:

- exact runtime version, package manager and lockfile;
- rules a reviewer could enforce, not slogans;
- the real test runner, test location and the failing-test-first rule;
- the genuine security footguns of that ecosystem and its audit command;
- the directory layout and which way dependencies point;
- build, test, lint and format commands that actually work in a fresh
  repository — these flow into both the generated CLAUDE.md and CI.

## 4. Update the tests

In `internal/lang/lang_test.go`:

- add the id to `wantStable`, removing it from `wantPlanned` if present;
- add its aliases to the alias table;
- add its primary option and expected default to `TestOptionsExposeDefaults`.

The existing table tests then enforce completeness automatically.

## 5. Verify

```
scripts/check.sh
```

That is the validation flow in its canonical order — gofmt, `go vet ./...`,
`go test ./...`, `golangci-lint run`, `go build ./...`. Then exercise the new
plugin through the CLI:

```
go run ./cmd/claude-repo-factory languages
go run ./cmd/claude-repo-factory new demo -l <id> --yes
```

Paste the real output of the last two so the new plugin can be seen working.

## 6. Report

Confirm explicitly that no file outside `internal/lang` changed. If one did,
explain precisely what the plugin contract was missing, and propose an ADR in
`docs/adr/` rather than leaving the core widened by accident.
