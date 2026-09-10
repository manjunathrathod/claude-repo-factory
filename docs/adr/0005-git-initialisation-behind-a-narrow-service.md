# 0005. Git initialisation sits behind a narrow, fixed-argument service

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Feature 4 asks the factory to initialise a Git repository inside the directory
[0004](0004-output-directory-is-the-parent.md) taught it to create, and is
explicit about the boundaries: `git init` only — no commits, no remotes, no
GitHub repositories. It is equally explicit about how: a fixed executable with
fixed arguments, no shell, no user-controlled text reaching a shell, and a
validated working directory.

`internal/gitutil` already existed and already satisfied part of that. Its
`ExecRunner` builds commands with `exec.CommandContext` and an explicit
argument slice, and `ErrGitMissing` already distinguished an absent
executable. But `Client` also exposes `AddAll`, `Commit` and `AddRemote` —
operations this milestone must not perform — and nothing validated the
directory git was told to run in.

Two forces:

- **Starting a subprocess is the highest-risk thing the tool does.** Directory
  creation can at worst write somewhere unintended; a badly built command line
  can execute something unintended.
- **"Do not do X yet" is easy to state and easy to violate.** A rule kept by
  discipline is a rule that erodes; a rule kept by the type system does not.

## Decision

**A narrow front door, not a wider one.** `gitutil.Repository` exposes only
`Init`. `Client` keeps its fuller surface for the milestones that will need
it, but the generator is handed a `generator.Git` interface with a single
method, so commits and remotes are not reachable from the generation path at
all. The restriction is structural rather than a matter of remembering.

**The working directory is validated before git runs.**
`ValidateWorkingDirectory` requires an existing, absolute, non-symlink
directory. Absolute because a relative path resolves against the factory's own
working directory rather than the generated repository; `Lstat` rather than
`Stat` because a symlink would run git somewhere other than the directory that
was created and reported to the user.

**Errors describe the problem, never the environment.** A missing git yields
an instruction to install it or to pass `--no-git`. It does not print `PATH`,
the resolved executable path, or any environment variable.

**Refusing an existing repository.** `Init` fails when the directory is
already a work tree. The factory creates repositories; it does not adopt or
re-initialise them, and doing so silently would hide that the caller did not
expect one to be there.

**A requested git repository that cannot be created is an error.** When
`InitializeGit` is set and no Git client is wired, `Prepare` fails rather than
skipping. "I asked for a repository and did not get one" must never pass
quietly.

## Consequences

- `internal/gitutil` is the only package in the tool that starts a subprocess,
  and it now has two entry points with different audiences: `Repository` for
  generation, `Client` for later milestones.
- The security properties are pinned by tests rather than asserted in prose. A
  branch name of `main; rm -rf / #$(whoami)` is passed through and asserted to
  arrive as one unsplit argument; the missing-git message is asserted not to
  contain `PATH`.
- Integration tests run a real `git init` and skip when git is absent, so the
  suite still passes on a machine without it. The guarantee is about what ends
  up on a real disk, so a fake git would prove nothing here.
- `git` remains an optional dependency: `--no-git` produces the directory
  without it, and is what the missing-git error suggests.

## Alternatives considered

**Use `gitutil.Client` directly from the generator.** Rejected: it exposes
`Commit` and `AddRemote`, which this milestone must not perform. Narrowing the
interface at the consumer makes the boundary structural.

**Check for git once at start-up.** Rejected: it would fail a `--no-git` run
on a machine without git, and the check would be stale by the time it matters.
`Init` checks at the point of use.

**Shell out to `sh -c "git init ..."`.** Never considered seriously; it is the
exact failure this design exists to prevent, and is called out in CLAUDE.md
section 9.
