---
description: Run the full quality gate and report against the definition of done before a commit or pull request
allowed-tools: Read, Glob, Grep, Bash
---

Run the complete quality gate for `claude-repo-factory` and report honestly.

## 1. Run every check

Run the validation flow first, in its canonical order:

```
scripts/check.sh
```

That is gofmt, `go vet ./...`, `go test ./...`, `golangci-lint run`,
`go build ./...`, and it stops at the first failure. For a complete report you
want every result, so if it stops early, run the remaining steps individually
rather than reporting only the first failure:

```
gofmt -l .
go vet ./...
go test ./...
golangci-lint run
go build ./...
```

Then the gates that sit outside the flow:

```
go test -race ./...
go mod tidy && git diff --exit-code go.mod go.sum
```

If a tool is not installed, or the local toolchain cannot run `-race`, say so
plainly rather than skipping it silently or implying it passed.

## 2. Smoke test the CLI

```
go run ./cmd/claude-repo-factory --help
go run ./cmd/claude-repo-factory languages
go run ./cmd/claude-repo-factory new demo -l go -t cli --yes
```

Check that the help text is accurate, that the language list matches what is
registered, and that the plan output does not claim files were written.

## 3. Inspect the diff

```
git status --short
git diff
```

Look for: debugging leftovers, commented-out code, secrets, personal paths,
binaries, coverage files, and `.claude/settings.local.json`.

## 4. Report

Produce a table with one row per check: the command, pass or fail, and the
relevant output line.

Then the definition-of-done checklist from `CLAUDE.md` section 12, each item
marked done, not done or not applicable.

Finish with a one-line verdict: ready to ship, or the specific things that
must be fixed first. Never report a gate as passing that you did not run.
