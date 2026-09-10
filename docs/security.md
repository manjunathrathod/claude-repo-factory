# Security

## Threat model

`claude-repo-factory` runs on developer machines with write access to the
filesystem, shells out to `git`, and generates content that lands in other
people's repositories. Three things follow.

1. **All input is untrusted.** The repository name, target directory, remote
   URL and every `--set` option are attacker-influenced, even when the
   "attacker" is a mistyped command or a copy-pasted script.
2. **The output is a supply-chain artifact.** Whatever the factory writes
   becomes someone else's CI configuration and someone else's dependency set.
3. **The tool is offline.** It makes no network calls. Adding one is a
   deliberate decision requiring an ADR.

## Rules

### Never execute a shell string

Every subprocess is built with `exec.CommandContext` and an explicit argument
slice. No `sh -c`, no interpolation of user input into a command line.
`internal/gitutil` is the only package that spawns a process, and
`TestCommitMessageIsPassedAsASingleArgument` pins the behaviour: a commit
message containing `$PATH`, quotes and `rm -rf /` reaches git as one inert
argument.

### Treat every path as hostile

- The confirmation summary is a security boundary: it is the block a user
  reads before agreeing. `config.ValidateNoControlCharacters` rejects newlines
  and escape sequences in every free-text field, so no value can forge a row
  in it or rewrite the screen. `cli.displayValue` quotes anything
  non-printable as defence in depth.
- `config.ValidateRemote` rejects a value git would read as an option, and the
  `ext::` and `fd::` transports, which execute a command.
- `config.ValidateOutputDirectory` rejects `..` segments, UNC and device paths,
  drive-relative paths such as `C:foo`, Windows reserved device names in any
  segment, and segments ending in a dot or a space.
- Repository names are validated in `config.ValidateProjectName` against
  `^[A-Za-z0-9][A-Za-z0-9._-]*$`, which excludes path separators, `..` and
  leading dashes.
- When generation lands, every resolved write path must be verified to stay
  inside the target directory: reject `..` traversal, absolute paths supplied
  as relative, and symlinked parents. A write that can escape the target
  directory is a critical defect.
- Never overwrite an existing non-empty directory without explicit
  confirmation. Clean up partial output on failure.

### Explicit, minimal file permissions

No world-writable files. No executable bit except on genuine scripts. Files
carry an explicit mode rather than inheriting whatever the umask gives.

### No secrets, anywhere

No tokens, keys, passwords, private URLs, personal paths or email addresses
in code, tests, fixtures, templates, documentation or commit messages.
Generated `.gitignore` files must exclude local secret files, and
`.claude/settings.local.json` is ignored in this repository.

### Templates

`text/template` does not escape output, so a template must never render user
input into a context where escaping matters. `missingkey=error` stays set:
a template referring to data that does not exist must fail loudly rather than
emit an empty value into someone's configuration.

### Dependencies

The set is Cobra, survey and the standard library. Any addition is a decision
that belongs in an ADR. `govulncheck ./...` must be clean before a release,
and CI runs it on every push.

### Generated CI

Workflows written into other repositories must pin action versions, declare
least-privilege `permissions`, and never echo secrets. A generated pipeline
that leaks a token is our defect, not the user's.

## Reviewing a change

Run the `security-reviewer` agent, or work through this list by hand:

- [ ] No new subprocess outside `internal/gitutil`, and no shell string.
- [ ] Every new path is validated and confined to the target directory.
- [ ] No new dependency, and no network call.
- [ ] No secrets, personal paths or private URLs in the diff.
- [ ] File modes are explicit and minimal.
- [ ] `missingkey=error` still set in `internal/render`.
- [ ] `govulncheck ./...` is clean.

## Reporting a vulnerability

Do not open a public issue. Report privately to the maintainers with a
description of the problem, the affected version and a reproduction. Describe
the class of problem rather than publishing a working exploit.
