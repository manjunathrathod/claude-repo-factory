# 0003. Type the project configuration model and validate it against the registry

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Feature 1 asks for a strongly typed project configuration model with nine
fields — project name, description, language, project type, package manager,
output directory, and toggles for Git, Claude configuration and GitHub
Actions — with safe validation, sensible defaults, and validation kept
separate from CLI concerns.

Eight of those nine already exist on `spec.Spec` under different names. The
ninth, package manager, does not exist anywhere. So the real question is not
"what shape should the new model be" but "does the factory grow a second
model, and how does a typed model stay ignorant of language names".

Three forces pull against each other:

- **ADR 0001** forbids the core knowing a language name. A frozen
  `const LanguageNode = "node"` in `internal/spec` would break that directly,
  and would also create a second source of truth alongside the registry that
  drifts the first time a plugin is added or renamed.
- **Strong typing** is the point of the request. `Language string` and
  `ProjectType string` accept any string, and the only thing standing between
  a typo and a generated repository today is `Definition.Validate`, which the
  CLI reaches only after it has already resolved the plugin.
- **Project types and package managers are declared per plugin** and have
  drifted: `service` means an HTTP service in four plugins that each describe
  it differently, `data` exists only in Python, and no plugin says anything
  about package managers even though `Instructions.Commands` hardcodes
  `npm ci` and `uv sync`.

## Decision

**`spec.Spec` is the project configuration model.** No second input model, no
rename of `internal/spec`. Feature 1 extends and hardens the type that already
exists, mapping the requested vocabulary onto it:

| Requested field | `spec.Spec` |
| --- | --- |
| ProjectName | `Name` |
| Description | `Description` |
| Language | `Language` |
| ProjectType | `ProjectType` |
| PackageManager | `PackageManager` *(new)* |
| OutputDirectory | `TargetDir` |
| InitializeGit | `Features.Git` |
| IncludeClaude | `Features.ClaudeConfig` |
| IncludeGitHubActions | `Features.GitHubActions` |

Go field names win over the requested labels: inside a type called `Spec`,
`ProjectName` and `OutputDirectory` stutter, and `Features` already groups the
toggles so that "professional by default" is one decision rather than nine.

**Three named string types carry the vocabulary.**

```go
type Language string
type ProjectType string
type PackageManager string
```

`Language` and `PackageManager` have **no constants in the core**. Their valid
values are whatever is registered. `ProjectType` has exactly four constants —
`api`, `cli`, `library`, `worker` — because a project type describes the shape
of a repository, not a language.

**Validity is asked of the registry, through an interface the core owns.**

```go
type Catalog interface {
    Languages() []Language
    Resolve(name string) (Language, bool)
    ProjectTypes(l Language) []ProjectType
    PackageManagers(l Language) []PackageManager
}

func (s Spec) Validate(c Catalog) error
```

`*plugin.Registry` implements `Catalog` — `plugin` already imports `spec`, so
the dependency direction is unchanged and no adapter type is needed.
`Validate` therefore rejects an unsupported language, project type or package
manager against the plugins actually compiled in, and adding Rust still
requires no core edit. A nil catalog is an error, not a skipped check.

**The Spec stores canonical ids, never aliases.** `node` is an alias of
`nodejs` and stays one: aliases are an input-resolution concern, resolved by
`Catalog.Resolve` before validation, so `--language node` produces
`Language: "nodejs"`. Two spellings of one value in a resolved Spec would
break equality, plan output and golden files. The friendly name the user sees
is `Descriptor.DisplayName`; `nodejs` is the id; `node` is one way to type it.

**`plugin.Descriptor` gains package managers and a closed project type.**

```go
type Descriptor struct {
    ID                    spec.Language
    DisplayName           string
    Summary               string
    Aliases               []string
    Status                Status
    ProjectTypes          []ProjectType          // ID is now spec.ProjectType
    DefaultProjectType    spec.ProjectType
    PackageManagers       []spec.PackageManager  // new
    DefaultPackageManager spec.PackageManager    // new
}
```

Package manager ids are language knowledge and live as constants beside the
plugin that owns them (`nodejs.go` declares npm, pnpm, yarn). The registry
extends its start-up validation: a stable plugin declares at least one package
manager, and both defaults must be members of their declared sets, so a
mis-wired plugin still fails at start-up rather than mid-generation.

The `plugin.ProjectType` struct survives so a plugin keeps its own display
text — "Cobra based command line tool" and "Command line tool published to
npm" are both true — but its `ID` is now drawn from the closed set. Package
managers get no such struct: `npm` and `poetry` describe themselves.

**Project types are remapped once:** `service` becomes `api` in all four
plugins, Java's default becomes `api`, and every current plugin also declares
`worker`. Python's `data` has no home in the four-value vocabulary and is
dropped.

**Path safety is a model rule, not a CLI rule.** `Validate` rejects reserved
and unsafe repository names (Windows device names, a trailing `.` or `-`) on
top of the existing pattern, and requires the final element of the cleaned
`TargetDir` to satisfy that same name pattern — which rejects `..`, `.`, a
bare separator and a filesystem root in one rule. Validation touches no
filesystem: existence and emptiness are the writer's business in milestone 2,
and confinement of `FileSpec.Path` remains the separate roadmap item it
already is.

## Consequences

**Good.** There is one model and one source of truth for what is supported;
the config model and the registry cannot drift because one asks the other.
Validation is callable without a terminal, a plugin lookup or a Cobra command,
which is what "separate from CLI concerns" means, and it is table-testable
against a fake catalog with no plugins registered. A typo in a project type is
now caught by the compiler in plugin code and by `Validate` in user input.
`Instructions.Commands` gains a legitimate place to branch — inside the
plugin, on a declared package manager — instead of hardcoding `npm ci`.

**Costs.** Every plugin file changes twice over: new descriptor fields, and
`service` renamed to `api`. `Spec.Validate` changes signature, so every caller
and test is touched. `spec` grows a small interface it did not have. Declaring
a package manager is not honouring one: `Instruct` still emits npm and uv
commands unconditionally, and making the commands follow the selected manager
is follow-on work, not part of this decision.

**What we live with.** Python loses `data`. That is real lost capability, and
the price of a shared vocabulary; reintroducing it means adding a fifth value
to the factory's project types deliberately, for every language, rather than
one plugin inventing an id. Equally, the day a plugin wants `notebook` or
`lambda`, that is a factory-level decision and an amendment here — a plugin
may narrow the set, never widen it.

**What this does not change.** No new dependency, no network call, no change
to the dependency direction, and no language name in the core: the four
project type constants name repository shapes, and the test still holds —
adding Rust edits one file in `internal/lang`.

## Alternatives considered

**A separate `ProjectConfig` that resolves into `Spec`.** The conventional
input-model/domain-model split. Rejected: it would be a near-clone of `Spec`
differing in one field, and the input layer already exists — `cli.newOptions`
holds the partially specified flags and `spec.Default()` is the resolution
step. A third representation between them adds a translation that carries no
new information and two validators that will disagree within a release.

**Rename `internal/spec` to `internal/config` and evolve it.** Buys a better
name and nothing else. It would mix a repository-wide mechanical rename with a
behaviour change in one commit, contradict the vocabulary quoted in ADR 0001
and ADR 0002, which are immutable, and churn every file in the project for a
noun. `Spec` is also the more accurate noun: it is the resolved description of
a repository, not the user's configuration file.

**Freeze the language enum in the core** (`const LanguageNode Language =
"node"`). Simplest and fully typed, and directly forbidden by ADR 0001. It
also puts the canonical list in two places — the const block and the registry
— so `languages` output and validation drift apart the first time they
disagree. The `Catalog` interface costs four method signatures and has two
real implementations from day one, the registry and a test fake, which is the
same shape as `prompt.Asker` and `gitutil.Runner`.

**A language-keyed map of package managers in the core**
(`map[Language][]PackageManager`). A language-name switch wearing a map's
clothing. Rejected for the same reason as the enum.

**Leave project types open, as they are today.** No migration and no lost
`data` type, but the vocabulary keeps drifting: four spellings of the same
concept already exist, and cross-language project-type file sets in milestone
3 would have nothing to key on.
