# Architecture decision records

One decision per file. An ADR records *why* something is the way it is, so
that a future reader can tell a deliberate choice from an accident.

## When to write one

Write an ADR when a change would:

- alter the `plugin.Language` contract;
- change the dependency direction between packages;
- add a third-party dependency;
- introduce a network call, telemetry, or anything the tool does not do today;
- change the layout or composition rules of a generated repository.

Adding a language does **not** need an ADR. That is the point of the design.

## Format

```markdown
# NNNN. Title in the imperative

- **Status:** Proposed | Accepted | Superseded by [NNNN](NNNN-....md)
- **Date:** YYYY-MM-DD

## Context
What forces are at play. What problem needs deciding.

## Decision
What we are doing, stated plainly.

## Consequences
What becomes easy, what becomes hard, what we now have to live with.

## Alternatives considered
What else was on the table and why it lost.
```

An accepted ADR is never edited. Supersede it with a new one and link both
ways.

## Index

| ADR | Title | Status |
| --- | --- | --- |
| [0001](0001-plugin-registry-for-language-support.md) | Plugin registry for language support | Accepted |
| [0002](0002-defer-repository-generation.md) | Defer repository generation to a second milestone | Accepted |
