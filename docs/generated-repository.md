# What a generated repository contains

> **Status:** this document describes the target for repository generation.
> Generation is not implemented yet — `new` resolves and validates a
> specification and prints the plan. See [roadmap.md](roadmap.md).

Every generated repository arrives with the scaffolding a professional team
would otherwise assemble by hand, plus the configuration that makes Claude
Code effective in it from the first commit.

## Contents

| Artifact | Purpose |
| --- | --- |
| Git repository | Initialised on the chosen default branch, with an optional `origin` remote |
| `README.md` | What the project is, how to build, test and run it |
| `CLAUDE.md` | The composed working contract — see below |
| `.claude/settings.json` | Permission allowlist tuned to the language toolchain |
| `.claude/agents/` | Specialist agents for the language and project type |
| `.claude/commands/` | Reusable feature, review and fix workflows |
| `.github/workflows/ci.yml` | CI running the language's build, test and lint commands |
| `.github/PULL_REQUEST_TEMPLATE.md` | Checklist tied to the definition of done |
| `.gitignore` | Language-appropriate, including local secret files |
| `docs/` | Architecture, coding standards, testing and security pages |
| `tests/` | Test structure matching the language's convention |
| Language configuration | `go.mod`, `pyproject.toml`, `package.json`, `pom.xml` and friends |
| Coding standards | Enforceable rules, not slogans |
| Security instructions | The real footguns of that ecosystem |
| Testing instructions | Runner, layout, coverage expectation, failing-test-first |
| Architecture instructions | Directory layout and dependency direction |
| Git instructions | Branch naming, commit format, PR rules |

## The composed CLAUDE.md

The generated CLAUDE.md is assembled from seven parts rather than rendered
from one template. The factory owns the ordering and the universal sections;
the language plugin fills in only what is genuinely language-specific.

| Part | Source |
| --- | --- |
| 1. Universal engineering instructions | The factory |
| 2. Language-specific instructions | `Instructions.Toolchain`, `Instructions.Standards` |
| 3. Project-type instructions | The selected `plugin.ProjectType` |
| 4. Testing instructions | `Instructions.Testing` |
| 5. Security rules | `Instructions.Security` |
| 6. Git rules | The factory |
| 7. Definition of done | The factory, plus the language's commands |

`Instructions.Commands` feeds both the CLAUDE.md command table and the
generated CI workflow, so the documentation and the pipeline cannot drift
apart.

## Language support

| Language | Status | Project types |
| --- | --- | --- |
| Node.js / TypeScript | Supported | api, cli, library, worker |
| Python | Supported | api, cli, library, worker |
| Go | Supported | api, cli, library, worker |
| Java | Supported | api, cli, library, worker |
| .NET | Planned | |
| Rust | Planned | |
| Terraform | Planned | |
| React | Planned | |
| Next.js | Planned | |

Run `claude-repo-factory languages` for the list this build actually
registers, which is always the authoritative answer.

## What is deliberately not included

- No licence beyond the chosen SPDX identifier — legal review is not ours.
- No cloud, container or deployment configuration. Deployment is
  organisation-specific and belongs in a separate template.
- No opinion on a branching model beyond trunk-based defaults.
- No telemetry, and nothing that phones home.
