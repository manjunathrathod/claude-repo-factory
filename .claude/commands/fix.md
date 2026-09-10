---
description: Diagnose and fix a bug, starting from a failing test that reproduces it
argument-hint: <bug description, error message or failing test>
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
---

Fix this in `claude-repo-factory`: **$ARGUMENTS**

The order below is not negotiable. A fix without a reproduction is a guess.

## 1. Reproduce

- Run the failing command or test and paste the real output.
- If there is no reproduction yet, write the smallest test that fails for the
  reported reason, and show it failing.
- If you genuinely cannot reproduce it, stop and report exactly what you
  tried and what happened instead. Do not "fix" code speculatively.

## 2. Diagnose

- Find the root cause, not the symptom. Read the call path from the entry
  point to the failure.
- State the cause in one sentence: which code, under which condition,
  produces which wrong result.
- Check whether the same mistake exists elsewhere in the repository. Say
  whether it does.

## 3. Fix

- Change the cause, not the symptom. Do not add a special case that hides it.
- Smallest change that fully resolves it. No unrelated refactor.
- If the root cause is an architecture violation — language knowledge in the
  core, a leaked dependency direction — fix it properly and say so.
- Never make a test pass by weakening the assertion, adding a skip or
  deleting the case.

## 4. Verify

Show the previously failing test now passing, then run the full gate and
paste the output:

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

## 5. Report

- The root cause in one sentence.
- What you changed, and why that is the right layer to change.
- The test that now guards against a regression.
- Any related occurrences you found and whether you fixed them.

Do not commit unless the user asks.
