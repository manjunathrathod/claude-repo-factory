# CLAUDE.md

Instructions for Claude Code working in this repository.

## 1. What this project is

`claude-repo-factory` is a CLI written in Go that creates fresh Git
repositories pre-configured for professional software development and for
Claude Code. A generated repository arrives with Git initialised, a README, a
composed CLAUDE.md, `.claude` configuration with specialist agents and
reusable workflows, GitHub Actions CI, a `.gitignore`, a docs structure, a
tests structure, language-specific configuration, and written standards for
coding, security, testing, architecture and Git.

**Current milestone.** The plugin contract, the registry, the typed project
configuration model (`config.ProjectConfig`), the CLI, the template engine and
the Git wrapper exist. Repository generation does not:
`Language.Files` returns `plugin.ErrNotImplemented` for every plugin, and
`new` prints the resolved plan instead of writing files. Do not quietly start
generating files as a side effect of another change.

**The governing constraint.** Adding a language must never require editing
core generator logic. If a change would make `internal/config`, `internal/cli`
or `internal/render` grow a `switch` on language name, the design is wrong.
Push the behaviour into the plugin.

## 2. Repository layout

```
cmd/claude-repo-factory/   Entry point. Thin: exit code translation only.
internal/cli/              Cobra commands, flag parsing, plan rendering.
internal/config/           ProjectConfig: the typed, validated repo description.
internal/plugin/           The Language contract and the registry. No language knowledge.
internal/lang/             One file per language plugin, plus the shared Definition.
internal/render/           text/template wrapper and naming helpers.
internal/gitutil/          Testable wrapper around the git command line.
internal/prompt/           Asker interface; survey implementation and a scripted one.
internal/version/          Build identity.
docs/                      Architecture, standards and decision records.
.claude/agents/            Specialist agents.
.claude/commands/          Reusable feature, review and fix workflows.
```

Dependencies point inward: `cli` depends on `lang` and `plugin`; `plugin`
depends only on `config`; `config` depends on nothing internal. Never invert
this.

## 3. The validation flow

**Every change is validated in this order, and only this order.**

```
gofmt
  |
  v
go vet ./...
  |
  v
go test ./...
  |
  v
golangci-lint run
  |
  v
go build ./...
```

Run the whole thing with one command:

```
scripts/check.sh          # or: make check
scripts/check.sh --skip-lint   # same flow without the slow lint pass
```

`scripts/check.sh` is the single definition of the order. The Makefile, CI and
every checklist in this repository defer to it, so the sequence cannot drift
between a developer machine and the pipeline. Do not re-order the steps in a
document, an agent or a workflow; change the script.

### Why this order

| Step | Why it sits here |
| --- | --- |
| 1. `gofmt` | Cheapest check. Formatting noise obscures every diff that follows, so it is settled first. |
| 2. `go vet ./...` | The first step that type-checks, so a compile break surfaces here rather than four steps later. Catches the mistakes that make test output confusing. |
| 3. `go test ./...` | Behaviour before style. A lint finding on code that does not work yet is noise. |
| 4. `golangci-lint run` | The broad, slow pass. Worth its runtime only once the code is known to be correct. |
| 5. `go build ./...` | Final gate: every package, including `cmd/`, compiles and links as a shippable artifact. |

Run each step individually while iterating, but the flow above is what
"validated" means. Do not report success without having run it, and never
claim a step passed that you did not run.

### Gates outside the flow

These extend the flow rather than reordering it. CI runs all of them.

| Gate | Command | Note |
| --- | --- | --- |
| Race detector | `go test -race ./...` | Unsupported on a 32-bit toolchain; CI covers it on Linux, macOS and Windows |
| Module tidiness | `go mod tidy` then `git diff --exit-code go.mod go.sum` | |
| Vulnerabilities | `make vulncheck` | `govulncheck ./...` |
| Coverage | `make cover` | Per-function report |

### Everything else

| Purpose | Command |
| --- | --- |
| Format in place | `gofmt -w .` |
| Build a binary | `make build` |
| Try the CLI | `go run ./cmd/claude-repo-factory languages` |

## 4. Universal engineering instructions

- **Understand before editing.** Read the surrounding package and follow its
  existing idioms rather than importing conventions from elsewhere.
- **Smallest change that fully solves the problem.** No speculative
  abstraction, no unrequested refactor riding along with a fix.
- **Finish the whole task.** If part of it is blocked, complete everything
  else and say plainly what was left and why.
- **Fail loudly, never silently.** No swallowed errors, no empty catch, no
  default that hides a missing value. `render` sets `missingkey=error` for
  exactly this reason.
- **No dead code.** Do not leave commented-out blocks, unused helpers or
  placeholder functions behind.
- **Comments explain why.** The code already says what. Comment the
  non-obvious constraint, the trade-off, the reason for the odd branch.
- **Deterministic output.** Anything a user or a test reads must be stable:
  sort map iteration, do not embed timestamps in generated content unless
  asked.
- **No new dependencies without a reason.** The dependency set is Cobra,
  survey and the standard library. Adding to it is a decision that belongs in
  an ADR, not in a commit.

## 5. Go instructions

- Go 1.25 or newer, as pinned by the `go` directive in `go.mod`. The floor
  is 1.25 because the patched `golang.org/x/sys` and `golang.org/x/text` that
  clear GO-2026-5024 and GO-2026-5970 require it.
- `gofmt` is the only formatter. Unformatted code fails CI.
- Every exported identifier has a doc comment starting with its own name.
  Every package has a package comment.
- Wrap errors with context: `fmt.Errorf("resolve target directory: %w", err)`.
  Never discard an error with `_`; `errcheck` runs with `check-blank`.
- Accept interfaces, return concrete types. Define an interface where it is
  consumed — `prompt.Asker` and `gitutil.Runner` are the pattern to follow.
- Any blocking or external call takes a `context.Context` first parameter.
- No package-level mutable state, with one deliberate exception: the language
  registry in `internal/lang`, which is populated by `init` and guarded by a
  mutex. Do not add a second exception.
- Prefer table-driven code over repetition; prefer clarity over cleverness.
- Keep `main` trivial. Logic that lives in `main` cannot be tested.

## 6. Project-type instructions: a CLI that generates repositories

- **Commands are thin.** A Cobra `RunE` resolves input and delegates. Business
  logic belongs in a package that a test can call directly.
- **Never write to `os.Stdout` from a command.** Use `cmd.OutOrStdout()` so
  tests can capture output.
- **Flags and prompts are two front doors to one function.** Every prompt has
  a flag equivalent, so the tool is fully scriptable. `--yes` must make any
  invocation non-interactive.
- **Errors are for the user, not the developer.** State what was wrong and
  what the valid options are — see `Registry.Get`, which lists the languages.
- **Exit codes matter.** 0 success, 1 failure, 130 user cancellation.
- **Generation must be safe.** When it lands: never write outside the target
  directory, never overwrite an existing non-empty directory without explicit
  confirmation, and clean up partial output on failure.
- **Templates are data, not code.** A template must not decide policy; the
  plugin decides and passes the answer in.

## 7. Adding a language (the core workflow)

This is the workflow the whole design exists to protect. It touches exactly
one package.

1. Create `internal/lang/<language>.go`.
2. Declare a `*Definition` with its `plugin.Descriptor`: id, display name,
   summary, aliases, `plugin.StatusStable`, project types and a default
   project type that is one of them.
3. Declare `RequiredOptions` for the answers the language cannot do without
   (a module path, a package name), each with a prompt, help text and a
   default derived from the Spec.
4. Implement `Instruct` to return the CLAUDE.md fragments: toolchain,
   standards, testing, security, architecture, and the canonical commands.
5. Register it in an `init` function with `register(...)`.
6. Add the language to `wantStable` in `internal/lang/lang_test.go`. The
   existing table tests then enforce that it is completely described.

Nothing outside `internal/lang` changes. If you find yourself editing
`internal/cli` or `internal/config` to add a language, stop and reconsider: the
missing capability belongs on the `plugin.Language` interface, and changing
that interface is an ADR-level decision recorded in `docs/adr/`.

Promoting a planned language (`.NET`, Rust, Terraform, React, Next.js) is the
same workflow: move it out of `planned.go` into its own file and fill it in.

## 8. Testing instructions

- The standard `testing` package only. No assertion libraries.
- Table-driven tests are the default shape. Name each case after the
  behaviour, not after the input.
- Test behaviour through the public API of a package. Tests live in
  `package foo_test` so they cannot reach into unexported internals.
- Use `t.TempDir`, `t.Setenv` and `t.Cleanup`; never leave state behind.
- Never touch the network, and never require a TTY. Interactive code is
  tested through `prompt.Scripted`.
- Every bug fix begins with a failing test that reproduces the bug.
- Tests must pass under `-race`. Do not skip or delete a failing test to get
  a green run; fix the cause.
- What must stay covered: registry registration and rejection rules, Spec
  validation, every stable plugin being completely described, CLI flag and
  prompt resolution, and the naming helpers in `render`.

## 9. Security rules

- **Never execute a shell string.** Build commands with `exec.CommandContext`
  and an explicit argument slice, as `gitutil.ExecRunner` does. No
  interpolation of user input into a command line, ever.
- **Treat every path as hostile.** Repository names are validated against a
  strict pattern in `config.ValidateProjectName`; generation must additionally reject any
  resolved path that escapes the target directory.
- **No secrets in the repository.** No tokens, keys, credentials, personal
  paths or email addresses in code, tests, fixtures, docs or commit messages.
- **No telemetry, no network calls.** This tool runs offline. Adding a network
  call requires an explicit decision and an ADR.
- **Generated content is a supply chain.** Anything the factory writes into
  someone else's repository must be pinned, minimal and free of secrets. CI
  workflows use pinned action versions.
- **Dependencies stay minimal and audited.** `govulncheck ./...` must be clean
  before a release.
- **File permissions are explicit.** Never write world-writable files; never
  create a file with the executable bit unless it is a script.

## 10. Architecture instructions

- **The core never learns a language name.** `internal/config`, `internal/cli`
  and `internal/render` contain no `switch` on language. Language knowledge
  lives only in `internal/lang`.
- **Extend through the interface.** A new capability that every language
  needs is a new method or field on `plugin.Language`, added deliberately and
  recorded in an ADR — not a special case in the caller.
- **Data flows one way.** CLI resolves a `config.ProjectConfig` → the plugin reads it →
  the plugin returns files and instructions → the writer renders them. A
  plugin never calls back into the CLI and never mutates the Spec it is given.
- **Language-specific answers live in `ProjectConfig.Options`,** keyed by constants the
  plugin owns (`lang.OptGoModule`). The core moves them around without
  interpreting them.
- **Injectable boundaries.** Anything touching the terminal, the filesystem or
  a subprocess sits behind an interface with a test double.
- **A significant architectural decision gets an ADR** in `docs/adr/`, numbered
  and immutable once accepted.

## 11. Git instructions

- `main` is the default branch and is never committed to directly.
- Branch names: `feature/<slug>`, `fix/<slug>`, `docs/<slug>`,
  `refactor/<slug>`, `chore/<slug>`.
- Commits follow Conventional Commits: `feat:`, `fix:`, `docs:`, `test:`,
  `refactor:`, `chore:`, `ci:`. The subject is imperative and under 72
  characters. The body explains why.
- One logical change per commit. Never mix a refactor with a behaviour change.
- Never commit generated binaries, coverage output or `.claude/settings.local.json`.
- Do not commit or push unless the user asked for it. Do not amend or
  force-push a branch that has been shared.
- Never use `--no-verify` or otherwise bypass hooks.
- A pull request uses the template in `.github/PULL_REQUEST_TEMPLATE.md` and
  must be green in CI before review is requested.

## 12. Definition of done

A change is done only when all of the following are true.

- [ ] It does what was asked, completely — no silently reduced scope.
- [ ] The validation flow (section 3) passes end to end, in order:
      - [ ] 1. `gofmt` — `gofmt -l .` prints nothing.
      - [ ] 2. `go vet ./...` is clean.
      - [ ] 3. `go test ./...` passes, with new tests covering the new behaviour.
      - [ ] 4. `golangci-lint run` is clean, with no new `nolint` that lacks a reason.
      - [ ] 5. `go build ./...` succeeds.
- [ ] `go test -race ./...` passes, or it is stated that the local toolchain
      cannot run it and CI will.
- [ ] `go mod tidy` leaves `go.mod` and `go.sum` unchanged.
- [ ] Every new exported identifier has a doc comment.
- [ ] Adding or changing a language required no edit to a core package, or an
      ADR explains why the interface had to change.
- [ ] `README.md`, this file and `docs/` reflect the new behaviour.
- [ ] The diff contains no secrets, no personal paths and no debugging leftovers.
- [ ] The user has been told plainly what was verified and what was not.

## 13. Working agreements

- If a request is ambiguous in a way that changes the outcome, ask. Otherwise
  choose the obvious default, state it and continue.
- Report what you actually ran and what it actually printed. If a test fails,
  say so and show the output.
- Do not weaken a lint rule, delete a test or lower a threshold to turn a
  build green. Fix the cause or explain why the rule is wrong.

## 14. Specialist agents and when to delegate

This repository ships project-level agents in `.claude/agents/`, listed with
selection guidance in [`.claude/agents/README.md`](.claude/agents/README.md).

| Agent | Owns |
| --- | --- |
| `architect` | Module boundaries, extensibility, the plugin contract, language and template strategy, ADRs |
| `typescript-developer` | The `nodejs` plugin and the TypeScript configuration generated repositories receive |
| `go-engineer` | Implementing and refactoring the factory itself |
| `language-plugin-author` | Adding or changing a plugin in `internal/lang` |
| `test-engineer` | Unit tests, integration tests, generated repository validation |
| `security-reviewer` | Unsafe filesystem operations, command injection, path traversal, secrets, unsafe shell execution |
| `code-reviewer` | Architecture, code quality, maintainability, regressions |
| `debugger` | Root-cause investigation before any code changes |
| `documentation-agent` | README, architecture docs, CLI usage, adding-a-language guide |
| `cli-ux-designer` | Command and flag naming, help text, prompts, error messages, output |

### Use a subagent when the work is independent or benefits from isolated context

Delegate when at least one of these is true:

- **Independent.** It does not depend on what you are doing right now, and its
  result merges in afterwards without coordination.
- **Better in isolated context.** A broad security sweep, a full-branch review
  or a search across many files, where the intermediate reading is noise the
  main thread does not need.
- **A distinct role with its own standards.** A review, an audit or a
  root-cause investigation is better done without the assumptions of whoever
  wrote the code.
- **Parallelisable.** Several genuinely separate pieces that do not touch the
  same files.

### Handle simple edits directly

Do **not** spawn an agent for:

- a small, well-understood edit — a rename, a typo, a flag description, one
  more case in an existing test table;
- work you already have the context for, where the agent would spend its first
  steps rediscovering what you just read;
- a single fix whose cause you already know;
- anything where describing the task takes longer than doing it.

Delegation costs a cold start: a subagent begins with none of this
conversation and re-derives context you already hold. Spend that cost only
when it buys something back. Never delegate to avoid understanding a problem
yourself, and never spawn several agents on the same question hoping one gets
it right — one agent per distinct piece of work.

A subagent's report is not shown to the user. Relay what matters, and never
state a pending agent's results before they have arrived.
