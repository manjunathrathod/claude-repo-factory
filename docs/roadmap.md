# Roadmap

## Milestone 1 — foundation (complete)

The plugin contract and everything around it, with no generation.

- [x] Professional Go project structure (`cmd/`, `internal/`, one concern per package)
- [x] `plugin.Language` contract and a concurrent-safe registry
- [x] `spec.Spec`: the resolved, language-agnostic repository description
- [x] Language plugins for Node.js/TypeScript, Python, Go and Java
- [x] Placeholders for .NET, Rust, Terraform, React and Next.js
- [x] Cobra CLI: `new`, `languages`, `version`
- [x] survey prompts behind an `Asker` interface, with a scripted double
- [x] `text/template` engine with naming helpers and `missingkey=error`
- [x] Testable `git` wrapper
- [x] Test suite covering registry, spec, plugins, CLI and rendering
- [x] `golangci-lint` configuration and GitHub Actions CI
- [x] CLAUDE.md, specialist agents, reusable workflows and documentation

## Milestone 2 — generation

Turn a validated `Spec` into files on disk.

- [ ] Filesystem writer with a dry-run mode and rollback on failure
- [ ] Path confinement: reject any write that escapes the target directory
- [ ] Embedded universal templates (README, `.gitignore`, PR template, docs)
- [ ] CLAUDE.md assembler composing the seven sections
- [ ] `Language.Files` implemented for the four supported languages
- [ ] Generated `.claude` configuration, agents and workflows
- [ ] Generated GitHub Actions CI driven by `Instructions.Commands`
- [ ] Git initialisation, first commit and optional remote
- [ ] Golden-file tests for every language and project type
- [ ] End-to-end test: generate, then run the generated project's own build

## Milestone 3 — breadth

- [ ] Promote .NET, Rust and Terraform from planned to supported
- [ ] React and Next.js, including the framework-within-a-language question
- [ ] Project-type-specific file sets, not just instructions
- [ ] `add` command to retrofit Claude configuration into an existing repository
- [ ] User-level configuration file for defaults (author, licence, organisation)

## Milestone 4 — extensibility beyond this binary

- [ ] External template packs resolved from a directory or a Git URL
- [ ] Organisation profiles: house standards layered over the defaults
- [ ] `doctor` command to validate a generated repository against its own rules

## Non-goals

These are deliberate exclusions, not omissions.

- Deployment, container and cloud configuration.
- Anything that requires a network call at generation time.
- Telemetry of any kind.
- A plugin system loading third-party binaries. Plugins are compiled in; that
  is what makes them reviewable.
