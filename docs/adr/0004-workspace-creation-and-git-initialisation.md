# 0004. The output directory is the parent; creation and git sit behind services

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Features 3 and 4 make the factory act on the filesystem for the first time: it
creates the destination directory, then initialises git inside it. Feature 3
states the expected result explicitly:

```
ProjectName     = payment-api
OutputDirectory = C:\Projects
                -> C:\Projects\payment-api
```

That contradicts what shipped in Feature 2, where `ResolvedOutputDirectory`
returned the repository directory itself, so `--dir C:\Projects` created the
repository *at* `C:\Projects`. The flag help, the prompt help and `docs/cli.md`
all documented that meaning.

The ambiguity was visible earlier and was resolved the other way: a review of
Feature 2 flagged that the help text read as "the parent" while the code meant
"the directory itself", and the response then was to make the documentation
match the code. Feature 3 settles it in the opposite direction, with a worked
example, so the decision is the user's rather than ours.

Three further forces:

- **These are the first genuinely dangerous operations in the tool.** Every
  prior milestone resolved and printed. Getting confinement wrong now means
  writing into a directory the user did not choose; getting subprocess
  construction wrong means command injection.
- **`internal/config` must stay free of side effects.** It validates strings;
  it does not know whether a path exists or whether git is installed.
- **`internal/gitutil` already existed** as a thin wrapper with a `Runner`
  interface, but exposed commit and remote operations that this milestone must
  not perform.

## Decision

**`OutputDirectory` is the parent.** `ResolvedOutputDirectory` returns it, and
`ResolvedProjectDirectory` returns it with `ProjectName` joined on. An empty
`OutputDirectory` means the working directory, so both readings agree on
`<cwd>/<ProjectName>` when the flag is omitted — the change is only visible
when `--dir` is given explicitly.

**Directory creation lives in `internal/filesystem`**, which imports nothing
internal. It takes a base directory and a single path segment, and knows
nothing about repositories, languages or configuration.

**Confinement is delegated to `os.Root`**, not reimplemented with string
comparison. A prefix check on a cleaned path is defeated by a symlink inside
the base and, on Windows, by the device namespace. `os.Root` holds a handle to
the base and rejects any name that escapes it.

**Git initialisation is a narrow front door.** `gitutil.Repository` exposes
only `Init`. The wider `Client` remains for later milestones, but the
generator cannot reach commits, remotes or anything networked through the
interface it is given.

**`internal/generator` owns the ordering rule**: validate, then create, then
init. It validates even though the CLI already has, because the CLI is not the
only possible caller and the guarantee has to hold for all of them.

## Consequences

- The meaning of `--dir` changed for anyone who used it. There is no
  deprecation path; the flag is one milestone old and unreleased.
- Three surfaces had to move together — flag help, prompt help and the
  prompt's *default*. Missing the default is exactly what produced a
  `payment-api/payment-api` nesting bug during implementation, caught by
  review and now pinned by a regression test.
- The summary block still shows the parent, because that is the question the
  user answered. The target is printed on its own line immediately above the
  confirmation prompt, so the gate never shows one path while creating another.
- The Go floor rose to 1.25.12. `os.Root` is the confinement mechanism, and
  GO-2026-4970 and GO-2026-4602 are root escapes in `os` fixed in 1.25.12 and
  1.25.8; a floor that allowed an unpatched toolchain would undermine the very
  guarantee this decision rests on.
- `internal/filesystem` duplicates rules `config` already enforces — reserved
  device names, traversal, control characters. That is deliberate: neither
  package should have to trust the other, and the duplication is only defence
  in depth once the filesystem copy is at least as strict as the config one.
- A configuration that asks for git when no git client is wired is an error,
  not a silent skip. "I asked for a repository and did not get one" must never
  pass quietly.

## Alternatives considered

**Keep `OutputDirectory` as the repository directory and split it.** The
generator would take the final segment as the name and its parent as the base.
This preserves Feature 2's meaning, but makes `--dir a/b/c` silently mean
"create `c` inside `a/b`" — the same join, done less visibly — and leaves the
flag's name at odds with every other tool that has an output directory.

**Reimplement confinement with `filepath.Clean` and a prefix check.** Rejected:
it is the standard way to get this wrong. It cannot see a symlink and needs
separate handling for Windows device and UNC paths. `os.Root` is stdlib and
does it at the syscall level.

**Use the existing `gitutil.Client` directly from the generator.** Rejected:
it exposes `Commit` and `AddRemote`, which this milestone must not perform.
Narrowing the interface at the consumer makes the restriction structural
rather than a matter of discipline.

**Put creation in `internal/config`.** Rejected: `config` imports nothing
internal and validates without touching the filesystem, which is what makes it
testable and reusable. Mixing in `os` calls would cost that.
