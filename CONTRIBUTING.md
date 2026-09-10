# Contributing

Thanks for working on `claude-repo-factory`. This document is the short
version; [`CLAUDE.md`](CLAUDE.md) is the full contract and applies to human
and AI contributors alike.

## Getting set up

```bash
git clone https://github.com/manjunathrathod/claude-repo-factory
cd claude-repo-factory
go mod download
make check
```

Requires Go 1.24 or newer. `golangci-lint` v2 is needed for `make lint`.

## The one rule that matters most

**Adding a language must not require changing core generator logic.**

`internal/spec`, `internal/cli`, `internal/render` and `internal/plugin`
contain no language names and no `switch` on one. If your change would add
either, the behaviour belongs in a plugin or on the `plugin.Language`
interface — and changing that interface needs an ADR.

## Common contributions

**Adding a language or framework.** Follow
[docs/adding-a-language.md](docs/adding-a-language.md). It touches one file
in `internal/lang` plus a test table. Confirm with `git diff --name-only`.

**Fixing a bug.** Start with a failing test that reproduces it. Fix the cause,
not the symptom.

**Changing the plugin contract.** Write an ADR in [docs/adr/](docs/adr/)
first: what is missing, what you propose, and what it costs every other
plugin.

## Before you open a pull request

Run the validation flow. It is one command, and it runs the steps in the
order this project standardises on:

```bash
scripts/check.sh    # or: make check
```

```
gofmt  ->  go vet ./...  ->  go test ./...  ->  golangci-lint run  ->  go build ./...
```

It stops at the first failure. `scripts/check.sh --skip-lint` drops the slow
step while you iterate, but the full flow must pass before you open the PR.

Then the gates that sit outside the flow:

```bash
go test -race ./...   # CI covers this if your toolchain cannot
go mod tidy           # must leave go.mod and go.sum unchanged
```

## Commits and branches

- Branch from `main`: `feature/<slug>`, `fix/<slug>`, `docs/<slug>`,
  `refactor/<slug>`, `chore/<slug>`.
- Conventional Commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`,
  `chore:`, `ci:`. Imperative subject under 72 characters; the body explains
  why.
- One logical change per commit. Never mix a refactor with a behaviour change.
- Never use `--no-verify`.

## Pull requests

Use [the template](.github/PULL_REQUEST_TEMPLATE.md) and satisfy the
definition of done in `CLAUDE.md` section 12. CI must be green before you
request review.

## Standards

- [docs/coding-standards.md](docs/coding-standards.md) — Go standards
- [docs/testing.md](docs/testing.md) — test strategy and house style
- [docs/security.md](docs/security.md) — threat model and security rules

Do not weaken a linter, delete a test or lower a threshold to get a green
build. Fix the cause, or make the case that the rule is wrong.

## Reporting a security issue

Do not open a public issue. See [docs/security.md](docs/security.md).
