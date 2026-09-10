# The command line interface

How the CLI is put together, what each command does, and the rules a change
to the command surface has to respect.

For user-facing usage — flags, examples, exit codes — see
[../README.md](../README.md). This document is about the design.

## Commands

| Command | Purpose |
| --- | --- |
| `create [name]` | Resolve, validate and confirm a project configuration. Aliased as `new`. |
| `languages` | List the language plugins in this build, with `--json` for scripts. |
| `version` | Build identity, with `--json`. |

The binary is `claude-repo-factory`. During development, run it from the
source tree with the command path:

```bash
go run ./cmd/claude-repo-factory create
```

The module has one entry point, `cmd/claude-repo-factory`, so `go run .` from
the repository root does not work. That is deliberate: a second `main` at the
root would be a second thing to keep in sync for no gain.

## What `create` does

```
flags + prompts  ->  config.ProjectConfig  ->  Validate  ->  summary  ->  confirm
```

1. `resolveConfig` starts from `config.Default()`, which enables every
   professional feature, and layers flags and prompt answers over it.
2. The language is resolved through the registry, which maps an alias such as
   `node` to the canonical id `nodejs`. Only the canonical id is stored.
3. The project type and package manager are offered from the **plugin
   descriptor**, never from a list in the CLI. This is what keeps
   `internal/cli` free of language knowledge.
4. `cfg.Validate(app.Registry)` runs the language-agnostic rules, then
   `language.Validate(cfg)` runs the language-specific ones.
5. Only after validation passes is the summary printed and the confirmation
   asked. A user is never asked to confirm a configuration that cannot work.

Repository generation is not implemented. On confirmation the command prints
`Configuration accepted.` and writes nothing.

## The nine questions

In order, matching the summary block:

| Question | Flag | Skipped when |
| --- | --- | --- |
| Project name | positional argument | a name was given |
| Description | `--description` | the flag was given |
| Programming language | `-l, --language` | the flag was given |
| Project type | `-t, --type` | the flag was given |
| Package manager | `--package-manager` | the flag was given, **or the language offers only one** |
| Output directory | `-d, --dir` | the flag was given |
| Initialize Git? | `--no-git` | the flag was given |
| Include Claude Code setup? | `--no-claude` | the flag was given |
| Include GitHub Actions? | `--no-ci` | the flag was given |

Languages may also declare required options of their own — a Go module path,
an npm package name. Those are asked after the nine, and satisfied by
`--set key=value`.

## Rules for changing the command surface

**Flags and prompts are two front doors to one function.** Every prompt must
have a flag equivalent, and `--yes` must make any invocation non-interactive.
A command that can only be driven interactively is a command that cannot be
scripted or tested.

**A flag always wins over its question.** For a string flag, "given" is
simply "non-empty". A boolean flag cannot express *unset* through its value,
so `newOptions.explicit` consults `cmd.Flags().Changed` instead — otherwise
`--no-git=false` would be indistinguishable from an absent `--no-git`, and the
command would either re-ask a question already answered or silently ignore
the answer.

**Never write to `os.Stdout`.** Use `cmd.OutOrStdout()`, so a test can capture
output. Everything a command needs — the registry, the asker, the writers —
arrives through `App`.

**The terminal sits behind `prompt.Asker`.** The CLI never calls survey
directly. Tests inject `prompt.Scripted`, which fails loudly on an unscripted
question so a test can never silently accept a default it did not intend.
`--yes` swaps in a `Scripted` with `UseDefaults`, which is why the same code
path serves both interactive and unattended runs.

**Errors are for the user.** State what was wrong and what the valid options
are. `config.FieldError` carries the field name and the offending value so the
message can point at the flag to fix.

**Exit codes:** `0` success — including declining the final confirmation,
which is a choice, not a failure — `1` failure, `130` cancellation. Ctrl+C at
any prompt surfaces as `prompt.ErrInterrupted`, which `Execute` translates to
130.

## Output

Two blocks, both deterministic so tests can assert on them:

- **The summary** (`internal/cli/summary.go`) is the nine confirmed values,
  always the same nine rows in the same order, with `-` for anything empty.
  The output directory is shown resolved to an absolute path, so the user
  confirms the path that would actually be written to rather than what they
  typed.
- **The plan** (`internal/cli/plan.go`, opt-in via `--plan`) additionally
  lists the artifacts generation would produce and the commands that would be
  recorded in CLAUDE.md and CI.

Display text for a language and a project type comes from the plugin
descriptor (`DisplayName`, `ProjectTypeDisplayName`). The CLI does not invent
its own casing or wording.

## Testing

Every test drives the real command tree through `run` in `cli_test.go`, with
a buffer for output and a scripted asker for input. No test needs a TTY.

What must stay covered: flag resolution, prompts being skipped when a flag was
supplied, the summary shape and ordering, package-manager choices following
the selected language, the defaults, cancellation, declining the
confirmation, and the error message for each class of invalid input.
