# 0001. Plugin registry for language support

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

The factory must support Node.js/TypeScript, Python, Go and Java now, and
.NET, Rust, Terraform, React, Next.js and others later. The list will keep
growing and will be extended by people who did not write the core.

The failure mode of every scaffolding tool is well known: language handling
starts as one `switch`, then another appears in the CLI, then a third in the
template layer. After five languages the core cannot be changed safely,
because every change risks four other languages.

The requirement was stated directly: adding a language later must not require
rewriting core generator logic.

## Decision

Language support is a plugin implementing a single interface:

```go
type Language interface {
    Descriptor() Descriptor
    Instructions(s spec.Spec) Instructions
    Files(s spec.Spec) ([]FileSpec, error)
    Validate(s spec.Spec) error
}
```

Supporting decisions:

- **`internal/plugin` knows no language.** It defines the contract and a
  registry, and imports only `internal/spec`.
- **All language knowledge lives in `internal/lang`,** one file per language,
  registered in `init`.
- **A declarative `Definition` implements the interface** so a plugin is data
  plus one function, not boilerplate.
- **Language-specific answers live in `Spec.Options`,** a `map[string]string`
  keyed by constants the plugin owns. A Go module path and a Maven groupId
  are indistinguishable to the core.
- **The registry validates at start-up:** duplicate ids, colliding aliases and
  an unknown default project type all fail immediately, via `MustRegister`.
- **Planned languages register with `StatusPlanned`,** so the roadmap is
  visible in `languages` output and the plugin shape is exercised by the CLI
  before it is filled in.
- **A table test enforces completeness:** every stable plugin must declare all
  five instruction sections, at least one command, a test command, and a
  default project type that exists.

## Consequences

**Good.** Adding a language touches one package and no core file; the
constraint is checkable with `git diff --name-only`. A mis-wired plugin fails
at start-up rather than mid-generation. The core stays small enough to test
exhaustively. The roadmap is machine-readable.

**Costs.** A capability that every language needs requires changing the
interface, which means changing every plugin — deliberately, and with an ADR.
The `Options` map is stringly typed, trading compile-time safety for a core
that never learns a language name; the plugin's `Validate` recovers the safety
at its own boundary. Plugins are compiled in, so extending the tool means
rebuilding it.

**What this forbids.** A `switch` on a language name in `internal/spec`,
`internal/cli`, `internal/render` or `internal/plugin`. Reviewers treat that
as a blocking finding.

## Alternatives considered

**A `switch` in the generator.** Simplest for four languages, unmaintainable
at ten, and directly contrary to the stated requirement.

**Configuration-driven languages (YAML or JSON manifests).** No recompilation
to add a language, but no type safety, no derived defaults, no logic in
`Validate`, and a much weaker review story — a manifest that writes files into
someone's repository deserves code review, not a data diff. Revisit for
*external template packs* in milestone 4, layered on top of this contract
rather than replacing it.

**Go plugins (`plugin` package) or external binaries.** Real runtime
extensibility, at the price of platform limitations, version-skew fragility
and loading unreviewed code that writes to the filesystem. Rejected on both
maintenance and security grounds.

**One interface per capability** (`FileProvider`, `InstructionProvider`, …).
More flexible, but every call site would need type assertions to discover
what a plugin supports. One interface with a completeness test is simpler and
catches gaps earlier.
