---
description: Review the current changes for correctness, architecture violations, security and test quality
argument-hint: [diff | staged | branch | path]
allowed-tools: Read, Glob, Grep, Bash
---

Review the changes in `claude-repo-factory`. Target: **$ARGUMENTS** (default:
the uncommitted working tree).

## Establish the diff

Choose the right command for the target and read the full diff before
commenting on any part of it:

```
git status --short
git diff                # working tree
git diff --staged       # staged
git diff main...HEAD    # branch
```

Read enough surrounding code that you can judge each hunk in context. A
finding based only on the diff text is a guess.

## Review in this order

**1. Correctness.** Wrong logic, unhandled errors, nil dereferences, boundary
conditions, races. Every finding needs a concrete failing scenario: these
inputs, this wrong result.

**2. Architecture.** Blocking findings even when the code works:
- language knowledge in `internal/config`, `internal/cli`, `internal/render` or
  `internal/plugin`;
- a `switch` on a language name anywhere in the core;
- inverted dependencies, new package-level mutable state;
- a plugin mutating the Spec it was handed;
- a change to `plugin.Language` with no ADR.

**3. Security.** Shell strings, unvalidated paths, writes that could escape
the target directory, secrets in the diff, a new dependency or network call,
generated CI without pinned actions or least-privilege permissions.

**4. Tests.** New behaviour without a test. Assertions on implementation
detail rather than behaviour. A loosened assertion, a new skip, a test that
needs a TTY or the network.

**5. Contract quality.** Missing doc comments, unwrapped errors, error
messages that do not say what to do next, `os.Stdout` instead of
`cmd.OutOrStdout()`, non-deterministic output.

**6. Simplification.** Duplication of something that already exists, an
abstraction with one caller, a helper that the standard library provides.

## Verify the claims

Run the checks yourself rather than assuming:

```
scripts/check.sh
```

That runs the validation flow in its canonical order — gofmt, `go vet ./...`,
`go test ./...`, `golangci-lint run`, `go build ./...`. If you only need a
quick signal, `scripts/check.sh --skip-lint` drops the slow step:

```
scripts/check.sh --skip-lint
```

## Report

- Findings ranked most severe first: file and line, one sentence on the
  defect, one on the consequence, and the concrete fix.
- State clearly which findings block a merge.
- Mark anything you could not confirm as needing confirmation.
- Finish with the definition-of-done checklist from `CLAUDE.md` section 12.

If the change is clean, say so in two sentences and stop. Do not manufacture
findings, and do not summarise the diff back to the author.
