# Documentation

Documentation for `claude-repo-factory`, the CLI that creates Git
repositories pre-configured for professional development and Claude Code.

## For users

| Document | What it covers |
| --- | --- |
| [../README.md](../README.md) | Installation, the commands, current status |
| [generated-repository.md](generated-repository.md) | What a generated repository contains and why |

## For contributors

| Document | What it covers |
| --- | --- |
| [architecture.md](architecture.md) | How the packages fit together and where the plugin boundary sits |
| [cli.md](cli.md) | The command surface, the `create` flow and the rules for changing it |
| [project-configuration.md](project-configuration.md) | The typed configuration model, its defaults and its validation rules |
| [adding-a-language.md](adding-a-language.md) | Step by step for a new language or framework plugin |
| [coding-standards.md](coding-standards.md) | The Go standards enforced in this repository |
| [testing.md](testing.md) | Test strategy, house style and what must stay covered |
| [security.md](security.md) | Threat model and the security rules for tool and output |
| [roadmap.md](roadmap.md) | Milestones and the languages still to come |
| [adr/](adr/) | Architecture decision records |

## For Claude Code

| Document | What it covers |
| --- | --- |
| [../CLAUDE.md](../CLAUDE.md) | The working contract: layout, commands, standards, definition of done |
| [../.claude/agents/README.md](../.claude/agents/README.md) | The ten specialist agents, and when to delegate rather than edit directly |
| [../.claude/commands/](../.claude/commands/) | Reusable feature, review, fix, add-language and ship-check workflows |

## Keeping documentation honest

If a document and the code disagree, the code is the truth and the document
is a bug. `docs/adding-a-language.md` in particular must stay in exact
agreement with `internal/lang/lang.go`.
