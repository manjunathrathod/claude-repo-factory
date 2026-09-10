# Specialist agents

Project-level agents for `claude-repo-factory`. Each file is a role with its
own instructions; Claude Code picks one from the `description` field, or you
name it directly.

## When to delegate, and when not to

**Delegation costs a cold start.** A subagent begins with no knowledge of this
conversation and has to re-derive context you already have. Spend that cost
only when it buys something back.

**Handle it directly — do not spawn an agent — when the work is:**

- a small, well-understood edit: renaming a symbol, fixing a typo, adjusting a
  flag description, adding a table case to an existing test;
- something you already have the context for, where the agent would spend its
  first steps rediscovering what you just read;
- a single obvious fix whose cause you already know;
- so short that describing the task takes longer than doing it.

**Delegate when the work is:**

- **independent** — it does not depend on what you are doing right now, and
  its result can be merged in afterwards without coordination;
- **better in isolated context** — a broad security sweep, a full-branch
  review, or a hunt through many files, where the intermediate reading is
  noise you do not want in the main thread;
- **a distinct role with its own standards** — a review, a security audit or a
  root-cause investigation benefits from arriving without the assumptions of
  whoever wrote the code;
- **parallelisable** — several genuinely separate pieces that do not touch the
  same files.

**Never** delegate to avoid understanding the problem yourself, and never
spawn several agents to look at the same thing hoping one gets it right. One
agent per distinct piece of work.

A subagent's report is not shown to the user. Relay what matters, and never
state a pending agent's results before they arrive.

## The agents

| Agent | Use it for |
| --- | --- |
| [architect](architect.md) | Module boundaries, extensibility, the plugin contract, language and template strategy, ADRs |
| [typescript-developer](typescript-developer.md) | The `nodejs` plugin and the TypeScript configuration generated repositories receive |
| [go-engineer](go-engineer.md) | Implementing and refactoring the factory itself, which is written in Go |
| [language-plugin-author](language-plugin-author.md) | Adding or changing a language plugin in `internal/lang` |
| [test-engineer](test-engineer.md) | Unit tests, integration tests, generated repository validation |
| [security-reviewer](security-reviewer.md) | Unsafe filesystem operations, command injection, path traversal, secrets, unsafe shell execution |
| [code-reviewer](code-reviewer.md) | Architecture, code quality, maintainability, regression risk |
| [debugger](debugger.md) | Root-cause investigation before any code changes |
| [documentation-agent](documentation-agent.md) | README, architecture docs, CLI usage, adding-a-language guide, ADRs |
| [cli-ux-designer](cli-ux-designer.md) | Command and flag naming, help text, prompts, error messages, output format |

## Which one, when the choice is not obvious

- **Design versus implementation.** `architect` decides where something goes
  and writes the ADR; `go-engineer` and `language-plugin-author` build it.
  If the answer is "it depends where this belongs", start with `architect`.
- **`typescript-developer` versus `go-engineer`.** The factory is written in
  Go. `typescript-developer` owns the TypeScript the factory *produces* —
  the `nodejs` plugin and the config generated repos receive. Anything about
  the CLI, registry, Spec or renderer is Go work.
- **`language-plugin-author` versus `typescript-developer`.** Use the plugin
  author for a new language in general; use the TypeScript developer when the
  language is TypeScript, React or Next.js and the ecosystem detail matters.
- **`debugger` versus `test-engineer`.** If you do not know why it fails, that
  is `debugger`. If you know the behaviour and need it pinned by a test, that
  is `test-engineer`.
- **`code-reviewer` versus `security-reviewer`.** Run both before a release.
  The code reviewer covers correctness, architecture and maintainability; the
  security reviewer covers only the threat model in `docs/security.md`, in
  more depth than a general review would.

## Reusable workflows

The commands in [`../commands/`](../commands/) — `/feature`, `/review`,
`/fix`, `/add-language`, `/ship-check` — are multi-phase procedures for the
main session. They can call these agents; use a workflow when you want the
whole procedure, and an agent when you want one role.

## Shared rules

Every agent follows [`../../CLAUDE.md`](../../CLAUDE.md). Three rules matter
enough to repeat:

1. The core never learns a language name. No `switch` on a language in
   `internal/spec`, `internal/cli`, `internal/render` or `internal/plugin`.
2. Validation runs in one fixed order — gofmt, `go vet ./...`,
   `go test ./...`, `golangci-lint run`, `go build ./...` — via
   `scripts/check.sh`. Report what you actually ran and what it actually
   printed. Never claim a step passed that you did not run.
3. Do not weaken a linter, delete a test or bypass a hook to get a green
   build.
