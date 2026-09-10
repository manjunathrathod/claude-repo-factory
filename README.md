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
ID         NAME                  STATUS   PROJECT TYPES                SUMMARY
dotnet     .NET                  planned  -                            C# on .NET 9 with xUnit and Roslyn analyzers
go         Go                    stable   cli, library, service        Go with golangci-lint, table-driven tests and a cmd/internal layout
java       Java                  stable   library, service, cli        Java 21 with Maven, JUnit 5, Spotless and SpotBugs
nextjs     Next.js               planned  -                            Next.js App Router with TypeScript and Playwright
nodejs     Node.js / TypeScript  stable   cli, library, service        TypeScript on Node.js with ESLint, Prettier and Vitest
python     Python                stable   cli, library, service, data  Python 3.12 with uv, ruff, mypy and pytest
react      React                 planned  -                            React single page app with Vite, TypeScript and Testing Library
rust       Rust                  planned  -                            Rust with clippy, rustfmt and cargo-deny
terraform  Terraform             planned  -                            Terraform modules with tflint, tfsec and Terratest
```

**Create a repository interactively.**

```bash
claude-repo-factory new
```

**Or fully from flags, with no prompts.**

```bash
claude-repo-factory new widget \
  --language go \
  --type service \
  --description "Widget control plane" \
  --author "Platform Team" \
  --license Apache-2.0 \
  --set go_module=github.com/acme/widget \
  --yes
```

Every prompt has a flag equivalent, and `--yes` makes any invocation
unattended, so the tool is fully scriptable.

### Flags for `new`

| Flag | Purpose |
| --- | --- |
| `-l, --language` | Language plugin, by id or alias (`ts`, `golang`, `py`, `jvm`) |
| `-t, --type` | Project type within the language (`cli`, `library`, `service`, …) |
| `-d, --dir` | Target directory (defaults to the repository name) |
| `--description` | One line description |
| `--author` | Author or owning team |
| `--license` | SPDX identifier, or `none` |
| `--branch` | Initial branch name (default `main`) |
| `--remote` | Git remote URL to register as `origin` |
| `--set key=value` | Language-specific option, repeatable |
| `-y, --yes` | Accept defaults for anything not given as a flag |
| `--no-git`, `--no-ci`, `--no-docs`, `--no-claude-workflows` | Turn features off |

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
