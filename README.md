# claude-repo-factory

A CLI that creates fresh Git repositories already configured for professional
software development and for Claude Code.

Instead of copying scaffolding between projects and re-explaining your
standards to every new repository, generate one that arrives with CI, docs,
tests, coding standards, security rules and a composed `CLAUDE.md` from the
first commit.

> **Status: milestone 1 of 4.** The plugin architecture, CLI, validation and
> planning are complete and tested. **Repository generation is not
> implemented yet** — `new` resolves your specification and prints the plan it
> would carry out. See [docs/roadmap.md](docs/roadmap.md).

## Install

```bash
go install github.com/manjunathrathod/claude-repo-factory/cmd/claude-repo-factory@latest
```

Or build from source:

```bash
git clone https://github.com/manjunathrathod/claude-repo-factory
cd claude-repo-factory
make build          # produces bin/claude-repo-factory
```

Requires Go 1.24 or newer.

## Use

**List the languages this build supports.**

```bash
claude-repo-factory languages
```

```
ID         NAME                  STATUS   PROJECT TYPES              PACKAGE MANAGERS  SUMMARY
dotnet     .NET                  planned  -                          -                 C# on .NET 9 with xUnit and Roslyn analyzers
go         Go                    stable   api, cli, library, worker  gomod             Go with golangci-lint, table-driven tests and a cmd/internal layout
java       Java                  stable   api, cli, library, worker  maven, gradle     Java 21 with Maven, JUnit 5, Spotless and SpotBugs
nextjs     Next.js               planned  -                          -                 Next.js App Router with TypeScript and Playwright
nodejs     Node.js / TypeScript  stable   api, cli, library, worker  npm, pnpm, yarn   TypeScript on Node.js with ESLint, Prettier and Vitest
python     Python                stable   api, cli, library, worker  uv, pip, poetry   Python 3.12 with uv, ruff, mypy and pytest
react      React                 planned  -                          -                 React single page app with Vite, TypeScript and Testing Library
rust       Rust                  planned  -                          -                 Rust with clippy, rustfmt and cargo-deny
terraform  Terraform             planned  -                          -                 Terraform modules with tflint, tfsec and Terratest
```

**Create a repository interactively.**

```bash
claude-repo-factory create
```

The command asks nine questions, then prints the configuration and asks you
to confirm it:

```
? Project name: payment-api
? Description: Payment service
? Programming language: Node.js / TypeScript
? Project type: API
? Package manager: npm
? Output directory: services/payment-api
? Initialize Git? Yes
? Include Claude Code setup? Yes
? Include GitHub Actions? Yes

Project Configuration
---------------------
Name: payment-api
Description: Payment service
Language: Node.js / TypeScript
Type: API
Package Manager: npm
Output Directory: C:\Projects\services\payment-api
Initialize Git: Yes
Claude Code Setup: Yes
GitHub Actions: Yes

? Create this project? Yes
Configuration accepted.
```

The package manager question is asked only when the language offers a choice:
Go has one, so it is skipped; Node.js, Python and Java have several.

> **This milestone stops there.** `create` resolves, validates and confirms a
> configuration. It writes no files and initialises no Git repository.

**Or fully from flags, with no prompts.**

```bash
claude-repo-factory create widget   --language go   --type api   --description "Widget control plane"   --dir services   --set go_module=github.com/acme/widget   --yes
```

Every prompt has a flag equivalent, and `--yes` makes any invocation
unattended, so the tool is fully scriptable. `create` is also available as
`new`.

Press Ctrl+C at any prompt to cancel; the command exits with status 130 and
writes nothing.

### Flags for `create`

| Flag | Purpose |
| --- | --- |
| `-l, --language` | Language plugin, by id or alias (`node`, `ts`, `py`, `golang`) |
| `-t, --type` | Project type: `api`, `cli`, `library` or `worker` |
| `--package-manager` | Package manager for the language, such as `npm`, `uv` or `maven` |
| `-d, --dir` | Target directory (defaults to the project name) |
| `--description` | One line description |
| `--author` | Author or owning team |
| `--license` | SPDX identifier, or `none` (default `MIT`) |
| `--branch` | Initial branch name (default `main`) |
| `--remote` | Git remote URL to register as `origin` |
| `--set key=value` | Language-specific option, repeatable |
| `-y, --yes` | Accept defaults for anything not given as a flag; never prompt |
| `--plan` | Also print the full generation plan: artifacts and commands |
| `--no-git` | Do not initialise a Git repository |
| `--no-claude` | Do not generate Claude Code configuration |
| `--no-ci` | Do not generate GitHub Actions workflows |
| `--no-docs`, `--no-claude-workflows` | Turn the remaining features off |

A `--no-*` flag always wins over its question: pass it and you are not asked.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success, including declining the final confirmation |
| `1` | Invalid configuration or another failure |
| `130` | Cancelled at a prompt with Ctrl+C |

### During development

Run the CLI straight from the source tree:

```bash
go run ./cmd/claude-repo-factory create
go run ./cmd/claude-repo-factory languages
```

The module has a single entry point, `cmd/claude-repo-factory`, so the
command path is required; `go run .` from the repository root will not work.

## What a generated repository will contain

Git initialisation, `README.md`, a composed `CLAUDE.md`, `.claude`
configuration with specialist agents and reusable workflows, GitHub Actions
CI, `.gitignore`, a docs structure, a tests structure, language-specific
configuration, and written coding, security, testing, architecture and Git
instructions with a pull request template.

The generated `CLAUDE.md` is assembled from seven parts — universal
engineering instructions, language-specific instructions, project-type
instructions, testing, security, Git rules and a definition of done — rather
than rendered from one template.

Full detail: [docs/generated-repository.md](docs/generated-repository.md).

## Design

The core never learns a language name. `internal/config`, `internal/cli`,
`internal/render` and `internal/plugin` contain no reference to Go, Python,
Node.js or Java; all language knowledge lives in `internal/lang`, behind a
single `plugin.Language` interface.

Adding a language means adding one file to `internal/lang` and one line to a
test table. Nothing in the core changes — and `git diff --name-only` proves it.

- [docs/architecture.md](docs/architecture.md) — how the pieces fit
- [docs/project-configuration.md](docs/project-configuration.md) — the typed configuration model and its validation
- [docs/adding-a-language.md](docs/adding-a-language.md) — the contributor workflow
- [docs/adr/0001](docs/adr/0001-plugin-registry-for-language-support.md) — why it is built this way

## Development

Every change is validated in one fixed order:

```
gofmt  ->  go vet ./...  ->  go test ./...  ->  golangci-lint run  ->  go build ./...
```

One command runs the whole flow, stopping at the first failure:

```bash
scripts/check.sh              # or: make check
scripts/check.sh --skip-lint  # same flow without the slow lint pass
```

`scripts/check.sh` is the single definition of that order; the Makefile, CI
and every checklist defer to it, so the sequence cannot drift between a
developer machine and the pipeline.

```bash
make test       # go test ./...
make race       # go test -race ./...
make lint       # golangci-lint run
make cover      # coverage per function
make build      # bin/claude-repo-factory
make vulncheck  # govulncheck ./...
```

CI runs the flow in the same order on every push, then the gates that sit
outside it: `-race` on Linux, macOS and Windows, module tidiness,
cross-platform builds and `govulncheck`.

See [CONTRIBUTING.md](CONTRIBUTING.md) and
[docs/coding-standards.md](docs/coding-standards.md).

## Working with Claude Code

This repository configures itself for Claude Code:

- [`CLAUDE.md`](CLAUDE.md) — the working contract
- [`.claude/agents/`](.claude/agents/) — ten specialist agents:
  `architect`, `typescript-developer`, `go-engineer`,
  `language-plugin-author`, `test-engineer`, `security-reviewer`,
  `code-reviewer`, `debugger`, `documentation-agent`, `cli-ux-designer`.
  [The roster](.claude/agents/README.md) explains which to pick and when to
  handle a change directly instead.
- [`.claude/commands/`](.claude/commands/) — `/feature`, `/review`, `/fix`,
  `/add-language`, `/ship-check`

## Licence

MIT. See [LICENSE](LICENSE).
