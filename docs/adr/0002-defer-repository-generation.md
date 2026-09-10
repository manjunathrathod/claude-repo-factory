# 0002. Defer repository generation to a second milestone

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

The first milestone was scoped explicitly to exclude repository generation:
establish the project structure, the plugin contract, the CLI, the tests, the
linting, the CI and the Claude Code configuration first.

That leaves an awkward question. `plugin.Language` declares
`Files(spec.Spec) ([]FileSpec, error)`, but no plugin can implement it yet.
Something has to stand in its place, and the choice matters: a scaffolding
tool that quietly does nothing is worse than one that refuses.

## Decision

Declare the full contract now and fail honestly.

- `plugin.ErrNotImplemented` is an exported sentinel. Every built-in plugin
  returns it from `Files`, wrapped with its own id.
- `new` resolves and validates the specification, prints the plan it would
  carry out, and then states plainly that generation is not implemented. It
  exits 0: nothing failed, and the user got the answer the tool can currently
  give.
- The plan output is real work, not a placeholder: it is how a user discovers
  what the tool will produce, and it is asserted on by tests.
- `CLAUDE.md` states the current milestone at the top, so an agent working in
  this repository does not start generating files as a side effect of an
  unrelated change.

## Consequences

**Good.** The contract is exercised end to end — registry, CLI, prompts,
validation, plan rendering — before any file is written, so the shape is
proven against four real languages rather than one. Milestone 2 becomes an
implementation of an interface that already has callers and tests. Users are
never misled about what happened.

**Costs.** The tool is not yet useful for its headline purpose. `Files` is
dead weight on the interface until milestone 2, and `errors.Is` handling for
`ErrNotImplemented` in `internal/cli` is temporary code that must be removed
when generation lands.

**Removal criterion.** When every stable plugin implements `Files`, delete the
`ErrNotImplemented` branch in `internal/cli/new.go` and the corresponding
test, and update `CLAUDE.md`, `README.md` and
`docs/generated-repository.md`. The sentinel itself stays, for plugins added
later that are not yet complete.

## Alternatives considered

**Omit `Files` from the interface until milestone 2.** Avoids the dead method,
but the interface would then be redesigned under pressure with four plugins
already written against it. Declaring the shape early is what makes ADR 0001
verifiable.

**Have `new` exit non-zero.** Accurate in one sense — the repository was not
created — but a non-zero exit means "something went wrong", and nothing did.
It would also make the plan output useless in a script.

**Generate a minimal repository now and improve it later.** Tempting, and the
reason it was rejected is scope discipline: a half-generated repository sets a
precedent for what "generated" means, and undoing that costs more than the
delay.
