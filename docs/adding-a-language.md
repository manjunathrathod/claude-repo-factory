# Adding a language

Adding a language to `claude-repo-factory` touches exactly one package:
`internal/lang`. If you find yourself editing anything else, stop — see
[When the contract is not enough](#when-the-contract-is-not-enough).

## 1. Read the pattern

- `internal/lang/lang.go` — the `Definition` and `OptionSpec` shapes.
- `internal/lang/golang.go` — a plugin with one required option.
- `internal/lang/java.go` — a plugin with two.
- `internal/lang/planned.go` — check whether your language is already listed
  as planned; if so, remove its entry as part of this change.

## 2. Create the plugin file

Create `internal/lang/<language>.go`:

```go
package lang

import (
	"strings"

	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// OptRustCrateName is the crate name for Rust repositories.
const OptRustCrateName = "rust_crate_name"

func init() { register(rust) }

var rust = &Definition{
	Meta: plugin.Descriptor{
		ID:          "rust",
		DisplayName: "Rust",
		Summary:     "Rust with clippy, rustfmt and cargo-deny",
		Aliases:     []string{"rs"},
		Status:      plugin.StatusStable,
		ProjectTypes: []plugin.ProjectType{
			{ID: "cli", DisplayName: "CLI", Summary: "Binary crate with clap"},
			{ID: "library", DisplayName: "Library", Summary: "Published crate"},
		},
		DefaultProjectType: "cli",
	},
	RequiredOptions: []OptionSpec{{
		Key:      OptRustCrateName,
		Prompt:   "Crate name",
		Help:     "Cargo crate name; lowercase with underscores.",
		Required: true,
		Default:  func(s spec.Spec) string { return pythonIdentifier(s.Name) },
	}},
	Instruct: func(s spec.Spec) plugin.Instructions {
		return plugin.Instructions{
			Toolchain:    strings.Join([]string{"- ..."}, "\n"),
			Standards:    strings.Join([]string{"- ..."}, "\n"),
			Testing:      strings.Join([]string{"- ..."}, "\n"),
			Security:     strings.Join([]string{"- ..."}, "\n"),
			Architecture: strings.Join([]string{"- ..."}, "\n"),
			Commands: []plugin.Command{
				{Name: "build", Run: "cargo build", Description: "Compile the crate"},
				{Name: "test", Run: "cargo test", Description: "Run the test suite"},
				{Name: "lint", Run: "cargo clippy -- -D warnings", Description: "Lint"},
				{Name: "format", Run: "cargo fmt", Description: "Apply formatting"},
			},
		}
	},
}
```

### Descriptor rules

| Field | Rule |
| --- | --- |
| `ID` | Lowercase, stable, used in `--language`. Never changes once released. |
| `Aliases` | Must not collide with another plugin or with any id. The registry rejects collisions at start-up. |
| `Status` | `plugin.StatusStable` once complete; `plugin.StatusPlanned` reserves the name. |
| `ProjectTypes` | At least two. Each needs an id, display name and summary. |
| `DefaultProjectType` | Must be one of the declared project types, or registration panics. |

### Option rules

Declare a `RequiredOptions` entry for every answer generation cannot proceed
without. Each needs a prompt, help text and a `Default` derived from the Spec,
so `--yes` produces a working repository with no interaction. Option keys are
exported constants the plugin owns; the core never interprets them.

## 3. Write instructions worth shipping

This text lands in someone else's CLAUDE.md and is read by both people and
Claude Code. Vague advice is worse than none.

| Section | What good looks like |
| --- | --- |
| `Toolchain` | Exact runtime version, package manager, lockfile, where configuration lives |
| `Standards` | Rules a reviewer can enforce: "no mutable default arguments", not "write clean code" |
| `Testing` | The runner, where tests live, naming, coverage expectation, failing-test-first |
| `Security` | Real footguns of that ecosystem — injection sinks, unsafe deserialisation — and the audit command |
| `Architecture` | The directory layout and which way dependencies point |
| `Commands` | Build, test, lint, format. They must work in a freshly generated repository. |

`Commands` feed both the generated CLAUDE.md and the generated CI, so an
inaccurate command breaks two things at once.

## 4. Update the tests

In `internal/lang/lang_test.go`:

- add the id to `wantStable`, removing it from `wantPlanned` if present;
- add each alias to the alias table;
- add the primary option and its expected default to
  `TestOptionsExposeDefaults`.

`TestStableLanguagesAreFullyDescribed` then enforces the rest automatically:
all five instruction sections present, at least one command, a test command,
and a valid default project type.

## 5. Verify

Run the validation flow, then exercise the plugin through the CLI:

```
scripts/check.sh
go run ./cmd/claude-repo-factory languages
go run ./cmd/claude-repo-factory new demo -l rust --yes
```

`scripts/check.sh` runs gofmt, `go vet ./...`, `go test ./...`,
`golangci-lint run` and `go build ./...` in that order, stopping at the first
failure. A new plugin frequently trips step 4 rather than step 3 — `goconst`
on repeated instruction strings is the usual one — so run the whole flow, not
just the tests.

## 6. Confirm the boundary held

`git diff --name-only` should list only files under `internal/lang` (plus
documentation). That is the test of the architecture, and it is worth
checking every time.

## When the contract is not enough

If a language genuinely cannot be expressed through `plugin.Language`, do not
widen the core quietly. Write an ADR in [adr/](adr/) describing what is
missing, what you propose adding to the interface, and what it costs every
other plugin. Change the contract deliberately, once, with a record — not
incrementally by special case.
