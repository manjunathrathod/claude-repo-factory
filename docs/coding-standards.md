# Coding standards

These are the standards enforced in this repository. They are also the
standards the factory advertises, so we hold ourselves to them first.

`CLAUDE.md` sections 4 and 5 are the short form. This document is the
reference.

## Formatting and tooling

Every change is validated by the flow in `CLAUDE.md` section 3, run with
`scripts/check.sh`:

```
gofmt  ->  go vet ./...  ->  go test ./...  ->  golangci-lint run  ->  go build ./...
```

| Step | Tool | Rule |
| --- | --- | --- |
| 1 | `gofmt` | The only formatter. `gofmt -l .` must print nothing. |
| 2 | `go vet` | Must be clean. First step that type-checks. |
| 3 | `go test` | The suite must pass. See [testing.md](testing.md). |
| 4 | `golangci-lint` | Must be clean against `.golangci.yml`. Do not weaken a rule to pass. |
| 5 | `go build` | Every package, including `cmd/`, must compile and link. |

`goimports` runs as part of step 4 and fixes import grouping: standard
library, third party, then `github.com/manjunathrathod/claude-repo-factory`.

Prose uses British spelling; technical identifiers keep their canonical form
(Maven `artifactId`, Java `serialization`), which is why `misspell` enforces
no locale.

A `nolint` directive needs a reason on the same line and a comment explaining
why the rule is wrong here. An unexplained `nolint` is a review blocker.

## Naming

- Packages: short, lowercase, singular nouns. No `util`, `common`, `helpers`,
  `base` or `misc`.
- Exported identifiers: no stutter. `plugin.Registry`, not `plugin.PluginRegistry`.
- Interfaces are named for what they do: `Asker`, `Runner`, `Language`.
- Test doubles say what they are: `Scripted`, `recorder`, `fake`.
- Acronyms keep their case: `ID`, `URL`, `HTTP`, `JSON`.

## Documentation comments

Every package has a package comment explaining what it is for and, where it
matters, why it exists at all. Every exported identifier has a doc comment
starting with its own name.

Comments explain *why*. The code already says what:

```go
// Parse compiles a template. Missing map keys are an error rather than
// silently rendering as <no value>, so a template referring to data that
// does not exist fails loudly during generation.
```

## Errors

- Wrap with context: `fmt.Errorf("resolve target directory: %w", err)`.
- Lowercase, no trailing punctuation, no "failed to" prefix.
- Sentinel errors are exported and compared with `errors.Is`:
  `plugin.ErrNotImplemented`, `gitutil.ErrGitMissing`.
- Never discard an error, including with `_`. `errcheck` runs with
  `check-blank`.
- Collect multiple validation failures with `errors.Join` and report them all
  at once. One error per run makes a user fix things one at a time.
- User-facing errors say what is wrong *and* what the valid options are:

```
unknown language "cobol" (available: dotnet, go, java, nextjs, nodejs, python, react, rust, terraform)
```

## Interfaces and dependencies

- Accept interfaces, return concrete types.
- Define the interface where it is consumed, not next to the implementation.
- Keep interfaces small. `prompt.Asker` has three methods; `gitutil.Runner`
  has one.
- Pass dependencies explicitly through constructors or a struct such as
  `cli.App`. No hidden globals.
- No package-level mutable state. The one exception is the registry in
  `internal/lang`, populated by `init` and mutex-guarded. Do not add a second.

## Concurrency

- Anything blocking or external takes a `context.Context` as its first
  parameter, named `ctx`.
- Shared state is either immutable or guarded. `plugin.Registry` uses an
  `RWMutex` because plugins are read from many places and written once.
- Tests must pass under `-race`.

## Determinism

Anything a user or a test observes must be stable:

- Sort before ranging over a map (`Spec.OptionKeys`, `Registry.List`).
- No timestamps in output unless the user asked for one.
- No dependence on filesystem ordering.

## Structure

- `main` only translates a result into an exit code.
- A Cobra `RunE` resolves input and delegates; logic lives in a package a test
  can call directly.
- Write to `cmd.OutOrStdout()`, never `os.Stdout`.
- No `switch` on a language name outside `internal/lang`. Ever.

## What not to do

- No speculative abstraction. An interface with one implementation and no
  second one in sight is a liability.
- No commented-out code, no unused helpers, no placeholder functions.
- No new dependency without an explicit decision. The set is Cobra, survey and
  the standard library.
- No `panic` in library code. `MustRegister` panics deliberately, at start-up,
  because a mis-wired plugin is a programming error.
