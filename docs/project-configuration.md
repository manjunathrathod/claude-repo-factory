# The project configuration model

`config.ProjectConfig` is the strongly typed, fully resolved description of a
repository the factory has been asked to create. It is the single value that
flows from the command layer into language plugins.

Two properties define it:

- **Validation lives with the model, not the CLI.** Any caller — a command, a
  test, or a future API — gets the same answer about whether a configuration
  is safe to act on.
- **The package names no language.** Which languages, project types and
  package managers exist is answered by a [Catalog](#the-catalog), which the
  plugin registry implements. See [ADR 0001](adr/0001-plugin-registry-for-language-support.md)
  and [ADR 0003](adr/0003-typed-project-configuration-model.md).

## Fields

```go
cfg := config.Default()
cfg.ProjectName = "widget"
cfg.Language = "go"
cfg.ProjectType = config.ProjectTypeAPI
cfg.PackageManager = "gomod"

if err := cfg.Validate(registry); err != nil {
    return err
}
```

| Field | Type | Default | Notes |
| --- | --- | --- | --- |
| `ProjectName` | `string` | — | Required. Becomes the directory name, so tightly constrained |
| `Description` | `string` | — | One line, used in README.md and CLAUDE.md |
| `Author` | `string` | — | Recorded in the licence and documentation |
| `License` | `string` | `MIT` | SPDX identifier, or `none` |
| `Language` | `Language` | — | Required. A canonical plugin id, never an alias |
| `ProjectType` | `ProjectType` | — | Required. `api`, `cli`, `library` or `worker` |
| `PackageManager` | `PackageManager` | — | Required. Valid values depend on the language |
| `OutputDirectory` | `string` | `ProjectName` | Where the repository is created |
| `DefaultBranch` | `string` | `main` | Initial branch |
| `Remote` | `string` | — | Optional origin URL |
| `InitializeGit` | `bool` | `true` | Initialise a Git repository |
| `IncludeClaude` | `bool` | `true` | Generate `.claude` configuration |
| `IncludeClaudeAgents` | `bool` | `true` | Generate specialist agents |
| `IncludeClaudeWorkflows` | `bool` | `true` | Generate reusable workflows |
| `IncludeGitHubActions` | `bool` | `true` | Generate CI |
| `IncludeDocs` | `bool` | `true` | Generate the docs structure |
| `IncludeTests` | `bool` | `true` | Generate the test structure |
| `IncludePRTemplate` | `bool` | `true` | Generate a pull request template |
| `IncludeCodingStandards` | `bool` | `true` | Generate coding standards |
| `IncludeSecurityPolicy` | `bool` | `true` | Generate security instructions |
| `Options` | `map[string]string` | empty | Language-specific answers |

Every toggle defaults to **on**. Flags only ever turn things off, so the
cheapest path produces the most complete repository.

`Options` is what keeps this package free of language knowledge: a Go module
path and a Maven group id are both just entries, keyed by constants the owning
plugin declares (`lang.OptGoModule`). The core moves them without interpreting
them.

## The typed vocabulary

```go
type Language string
type ProjectType string
type PackageManager string
```

All three are named string types, so a bare string cannot be passed where a
language is meant. Only one of them is a closed enum.

**`ProjectType` is closed** — `api`, `cli`, `library`, `worker`. It describes
the shape of a repository, not a language, so freezing it here costs nothing:
adding Rust still requires no change to this package. A plugin may declare a
*subset* of these; it may never invent a new one, and the registry rejects a
plugin that tries at start-up.

**`Language` and `PackageManager` have no constants here.** Freezing them
would put language names in the core, which ADR 0001 forbids, and would create
a second source of truth that drifts from the registry the first time a plugin
is added or renamed.

`Language` always stores a **canonical id**, never an alias. `node` is a way to
*type* `nodejs`; resolve it with `Catalog.Resolve` before assigning, which is
what the CLI does. Two spellings of one language reaching a resolved
configuration would break equality and plan output.

## The Catalog

```go
type Catalog interface {
    Languages() []Language
    Resolve(name string) (Language, bool)
    ProjectTypes(l Language) []ProjectType
    PackageManagers(l Language) []PackageManager
}
```

`*plugin.Registry` implements it, so support questions are answered by the
plugins actually registered in this build. `Languages` reports only stable
plugins: a planned language is announced in `languages` output but must never
pass validation.

A nil catalog is an **error**, not a skipped check — silently accepting any
language would turn a validation call into a no-op.

## Validation

`Validate` reports **every** problem at once, so a user fixes one round of
mistakes rather than rediscovering them one command at a time. The single
exception: once the language is missing or unsupported, the project type and
package manager checks are skipped, because both are language-relative and
would only add noise.

### Project name

A name becomes a directory name, part of a Git remote URL and part of
generated identifiers, so it admits only what is unambiguous in all three.

| Rule | Rejected example |
| --- | --- |
| Not empty | `""` |
| 64 characters or fewer | 65 `a`s |
| No leading or trailing whitespace | `" widget"` |
| Matches `^[A-Za-z0-9][A-Za-z0-9._-]*$` | `acme/widget`, `-widget`, `my widget`, `widget$(id)` |
| No `..` anywhere | `a..b` |
| No trailing dot | `widget.` |
| Not a Windows reserved device name | `nul`, `COM1`, `nul.txt` |

The pattern excludes every path separator, so **a valid name can never
contribute more than one path segment**. That is the property the traversal
defence rests on, and a test pins it. `..` is rejected separately as defence in
depth, so no name could read as a traversal component even if the pattern were
later widened.

The reserved-name list is exactly the documented set: `CON`, `PRN`, `AUX`,
`NUL`, `COM1`-`COM9`, `LPT1`-`LPT9`. `COM0` and `LPT0` are **not** reserved and
are accepted.

### Output directory

Empty is valid and means "a directory named `ProjectName`". Otherwise:

| Rule | Rejected example |
| --- | --- |
| No `..` segment, anywhere | `../etc`, `a/b/../c` |
| No NUL byte | `widget<NUL>/etc` |
| Not a filesystem root | `/`, `C:\` |

The `..` check runs on the **raw** slash-normalised input, not the cleaned
path. `path.Clean` resolves `..` away, and for an absolute path it *discards* a
`..` that would escape the root — so `/srv/../../etc` cleans to `/etc` and the
traversal disappears before it can be detected. The policy is therefore strict:
no `..` segment is accepted even when it would resolve back inside the base. A
scaffolding target has no legitimate need for one, and a rule with no
exceptions is the only kind that stays correct as callers change.

Validation is **filesystem-free**. Whether the directory exists, is empty, or
is writable is the writer's concern at generation time. Confining each
generated `FileSpec.Path` to the target directory remains a separate
milestone-2 requirement; this model bounds the target, not the files inside it.

### Errors

Every failure is a `*FieldError` wrapping one sentinel, so callers match a
category rather than message text:

```go
if errors.Is(err, config.ErrPathTraversal) { ... }

var fieldErr *config.FieldError
if errors.As(err, &fieldErr) {
    fmt.Println(fieldErr.Field, fieldErr.Value, fieldErr.Reason)
}
```

`ErrMissingField`, `ErrUnsafeName`, `ErrPathTraversal`, `ErrUnsupportedLanguage`,
`ErrUnsupportedProjectType`, `ErrUnsupportedPackageManager`, `ErrInvalidValue`.

Messages name the valid alternatives, because an error that only says "no"
makes the user guess:

```
ProjectType: "service": must be one of api, cli, library, worker
PackageManager: "npm": go supports gomod
```

## Two validation passes

| Pass | Owner | Checks |
| --- | --- | --- |
| `cfg.Validate(catalog)` | the model | Name and path safety, branch, and that the language, project type and package manager are supported |
| `language.Validate(cfg)` | the plugin | Only genuinely language-specific requirements — the options that plugin owns, such as a Go module path |

The split matters: the plugin no longer duplicates project-type checking, so
there is one rule in one place.

## Adding a field

1. Add it to `ProjectConfig` with a doc comment.
2. Give it a default in `Default` if "off" is the wrong starting point.
3. Add its rule to `Validate`, using an existing sentinel or adding one.
4. Add table cases to `internal/config/validate_test.go`, including the
   rejection path.
5. Surface it in `internal/cli/new.go` as **both** a flag and a prompt, and
   render it in `writePlan`.
6. Update the field table above.

If the field is language-specific, it does not belong here at all — it belongs
in `Options`, declared as an `OptionSpec` by the plugin that owns it. See
[adding-a-language.md](adding-a-language.md).
