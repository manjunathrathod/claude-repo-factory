# Architecture

## The problem this shape solves

A repository generator attracts one specific kind of rot: every new language
adds a branch to the generator, until the core is a pile of `switch` statements
that nobody can change safely. The whole structure below exists to make that
impossible.

**The rule:** the core never learns a language name. `internal/config`,
`internal/cli`, `internal/render` and `internal/plugin` contain no reference
to Go, Python, Node.js or Java. Language knowledge lives only in
`internal/lang`.

## Packages

```
cmd/claude-repo-factory   Entry point; translates a CLI result into an exit code.
        |
internal/cli              Cobra commands, flag parsing, plan rendering.
        |
        +-- internal/lang       Language plugins. One file per language.
        |         |
        +---------+-- internal/plugin   The Language contract and the registry.
        |                     |
        +---------------------+-- internal/config The typed repository description.
        |
        +-- internal/render     text/template wrapper and naming helpers.
        +-- internal/prompt     Asker interface; survey and scripted implementations.
        +-- internal/gitutil    Testable wrapper around the git command line.
        +-- internal/version    Build identity.
```

Dependencies point inward and never the other way:

- `config` imports nothing internal. It is the vocabulary everything shares.
- `plugin` imports only `config`. It defines the contract, knows no language, and
  implements `config.Catalog` so validation can ask what is registered.
- `lang` imports `plugin` and `config`. It is the only package with language
  knowledge.
- `cli` imports `lang`, `plugin`, `config` and `prompt`. It orchestrates.

## The three core types

**`config.ProjectConfig`** is the resolved, language-agnostic description of the
repository to create: name, description, author, license, target directory,
branch, remote, the chosen language and project type, feature toggles, and an
`Options` map. It carries no generation logic.

`Options` is the escape hatch that keeps the core language-free. A Go module
path and a Maven groupId are both just entries in a `map[string]string`,
keyed by constants the plugin owns (`lang.OptGoModule`). The core moves them
around without interpreting them.

**`plugin.Language`** is the extension point:

```go
type Language interface {
    Descriptor() Descriptor
    Instructions(c config.ProjectConfig) Instructions
    Files(c config.ProjectConfig) ([]FileSpec, error)
    Validate(c config.ProjectConfig) error
}
```

`Descriptor` is static metadata for listing and lookup. `Instructions` are the
language-specific fragments of the generated CLAUDE.md. `Files` returns what
to write. `Validate` enforces language-specific requirements on a Spec.

**`plugin.Registry`** resolves user input to a plugin. It is safe for
concurrent use, matches case-insensitively on ids and aliases, and rejects
duplicate ids, colliding aliases and an unknown default project type at
registration — so a mis-wired plugin fails at start-up, not mid-generation.

## Flow of a `new` command

1. `cli` starts from `spec.Default()`, which enables every professional
   feature. Flags and prompts only ever turn things off or fill things in.
2. `cli` resolves the language through the registry, then asks the plugin what
   language-specific options it needs (`lang.Options`) and collects them.
   The CLI never knows what those answers mean.
3. `spec.Validate` checks the language-agnostic invariants; `Language.Validate`
   checks the language-specific ones.
4. `cli` prints the plan.
5. `Language.Files` returns the files to write. **Today it returns
   `plugin.ErrNotImplemented` for every plugin** — generation is the next
   milestone.

## Injectable boundaries

Everything that touches the outside world sits behind an interface with a
test double, which is why the whole suite runs with no TTY, no network and no
subprocesses:

| Boundary | Interface | Real | Test double |
| --- | --- | --- | --- |
| Terminal | `prompt.Asker` | `prompt.Survey` | `prompt.Scripted` |
| Subprocess | `gitutil.Runner` | `gitutil.ExecRunner` | a recording fake |
| Output | `io.Writer` on `App` | `os.Stdout` | `bytes.Buffer` |

## Composing the generated CLAUDE.md

The generated CLAUDE.md is assembled, not templated as one blob:

1. universal engineering instructions (owned by the factory);
2. language-specific instructions (`Instructions.Toolchain`, `.Standards`);
3. project-type instructions (from the selected `ProjectType`);
4. testing instructions (`Instructions.Testing`);
5. security rules (`Instructions.Security`);
6. Git rules (owned by the factory);
7. definition of done (owned by the factory).

The factory owns the ordering and the universal sections; a plugin fills in
only what is genuinely language-specific. `Instructions.Commands` feed both
CLAUDE.md and the generated CI, so the two can never disagree about how the
project is built.

## Extending the contract

Adding a language changes one package. Adding a *capability* that every
language needs changes `plugin.Language`, and that is a deliberate,
recorded decision — see [adr/](adr/). A capability that only one language
needs is not a contract change; it belongs inside that plugin.
