# Testing

## Strategy

The whole suite runs in seconds, offline, with no TTY and no subprocesses.
That is a design property, not luck: every boundary that touches the outside
world sits behind an interface with a test double.

| Boundary | Interface | Test double |
| --- | --- | --- |
| Terminal | `prompt.Asker` | `prompt.Scripted` |
| Subprocess | `gitutil.Runner` | a recording fake in `git_test.go` |
| Output | `io.Writer` on `cli.App` | `bytes.Buffer` |

If a change makes the suite need the network, a terminal or a real `git`, the
change is wrong.

## House style

- The standard `testing` package only. No assertion or mocking libraries.
- External test packages (`package cli_test`) so tests exercise the public
  API and cannot depend on unexported internals.
- Table-driven by default. Name each case after the behaviour it pins down:
  `"branch with a space"`, not `"case 3"`.
- Failure messages show call, result and expectation:

```go
t.Errorf("Kebab(%q) = %q, want %q", in, got, want)
```

- `t.TempDir`, `t.Setenv`, `t.Cleanup`. Never leave state behind.
- Subtests with `t.Run` so a failure names itself.
- Assert on sentinel errors with `errors.Is`; assert on message text only for
  the part a user actually relies on.

## What must stay covered

These are the tests that hold the architecture together. Do not delete or
weaken them.

**`internal/plugin`** — registration rejects a nil language, an empty id, a
duplicate id, an alias claimed by another plugin, an alias colliding with an
id, and an unknown default project type. `List` is sorted; `Available`
excludes planned languages; concurrent lookup is race-free.

**`internal/config`** — every validation rule, plus the rule that all problems
are reported in one error rather than one per run. Option handling on a zero
Spec. Sorted option keys.

**`internal/lang`** — every promised language is registered with the right
status; every alias resolves; and `TestStableLanguagesAreFullyDescribed`
enforces that each stable plugin declares all five instruction sections, at
least one command, a test command, and a default project type that exists.
This table test is what makes a new plugin honest without anyone reviewing it
by hand.

**`internal/cli`** — each flag changes the resolved Spec; every prompt is
skipped when the equivalent flag was supplied; invalid input produces an error
naming the valid alternatives; the plan output does not claim files were
written.

**`internal/render`** — the naming helpers across every casing, including
acronyms such as `HTTPServer`, and that a missing template key is an error.

## Adding a test

1. Write it against the public API of the package.
2. Give the case a behavioural name.
3. Make it fail first — a test that has never failed proves nothing.
4. Keep it deterministic: no map iteration order, no clock, no host paths.

## Fixing a failing test

Reproduce, diagnose, then fix the cause. Never make a test pass by loosening
the assertion, adding `t.Skip` or deleting the case. If the test is right and
the code is wrong, fix the code and say so.

## Commands

```
scripts/check.sh               # the whole validation flow; tests are step 3
go test ./...                  # the suite on its own
go test -race ./...            # outside the flow; required before merge
go test ./internal/lang/ -run TestAliases -v
make cover                     # coverage per function
```

Tests are step 3 of the validation flow (`CLAUDE.md` section 3): gofmt,
`go vet ./...`, `go test ./...`, `golangci-lint run`, `go build ./...`. Vet
runs before the suite deliberately — it type-checks, so a compile break
surfaces as a vet failure rather than as confusing test output.

CI runs the suite with `-race` on Linux, macOS and Windows. A test that only
passes on one platform is a bug — usually a hard-coded path separator.
