package lang

import (
	"strings"

	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// OptGoModule is the module path for Go repositories.
const OptGoModule = "go_module"

func init() { register(golang) }

var golang = &Definition{
	Meta: plugin.Descriptor{
		ID:          "go",
		DisplayName: "Go",
		Summary:     "Go with golangci-lint, table-driven tests and a cmd/internal layout",
		Aliases:     []string{"golang"},
		Status:      plugin.StatusStable,
		ProjectTypes: []plugin.ProjectType{
			{ID: "cli", DisplayName: "CLI", Summary: "Cobra based command line tool"},
			{ID: "library", DisplayName: "Library", Summary: "Importable module with a public API"},
			{ID: "service", DisplayName: "HTTP service", Summary: "HTTP or gRPC service"},
		},
		DefaultProjectType: "cli",
	},
	RequiredOptions: []OptionSpec{{
		Key:      OptGoModule,
		Prompt:   "Go module path",
		Help:     "Fully qualified module path, for example github.com/acme/widget.",
		Required: true,
		Default:  func(s spec.Spec) string { return "github.com/example/" + strings.ToLower(s.Name) },
	}},
	Instruct: func(s spec.Spec) plugin.Instructions {
		return plugin.Instructions{
			Toolchain: strings.Join([]string{
				"- Go 1.24 or newer, pinned by the `go` directive in `go.mod`.",
				"- `gofmt` is the only formatter. Unformatted code fails CI.",
				"- `golangci-lint` runs with the ruleset in `.golangci.yml`; do not weaken it to make a lint pass.",
				"- Dependencies are added with `go get` followed by `go mod tidy` in the same commit.",
			}, "\n"),
			Standards: strings.Join([]string{
				"- Exported identifiers carry doc comments that start with the identifier name.",
				"- Errors are wrapped with `fmt.Errorf(\"context: %w\", err)`; never discard an error with `_`.",
				"- Accept interfaces, return concrete types. Define interfaces where they are consumed.",
				"- Every blocking call takes a `context.Context` as its first parameter.",
				"- No package-level mutable state. Pass dependencies explicitly through constructors.",
			}, "\n"),
			Testing: strings.Join([]string{
				"- Table-driven tests are the default shape; name subtests after the behaviour under test.",
				"- Use `t.TempDir`, `t.Setenv` and `t.Cleanup` rather than manual setup and teardown.",
				"- Tests must pass with `-race`. Do not skip tests to make CI green.",
				"- Every bug fix starts with a failing test that reproduces the bug.",
			}, "\n"),
			Security: strings.Join([]string{
				"- `govulncheck ./...` must be clean before a release.",
				"- Build commands with `exec.CommandContext` and explicit arguments; never pass a shell string.",
				"- Validate and clean every path derived from user input before touching the filesystem.",
				"- Use `crypto/rand` for anything security-relevant, never `math/rand`.",
			}, "\n"),
			Architecture: strings.Join([]string{
				"- `cmd/<binary>/main.go` stays thin: it wires dependencies and calls into `internal/`.",
				"- `internal/` holds implementation packages that must not be imported by other modules.",
				"- `pkg/` exists only when there is a deliberate, supported public API.",
				"- Package names are short, lowercase nouns. No util or common packages.",
			}, "\n"),
			Commands: []plugin.Command{
				{Name: "build", Run: "go build ./...", Description: "Compile every package"},
				{Name: "test", Run: "go test ./...", Description: "Run the full test suite"},
				{Name: "race", Run: "go test -race ./...", Description: "Run tests under the race detector"},
				{Name: "vet", Run: "go vet ./...", Description: "Run the standard static analysis"},
				{Name: "lint", Run: "golangci-lint run", Description: "Run the full lint suite"},
				{Name: "format", Run: "gofmt -w .", Description: "Format all Go sources"},
			},
		}
	},
}
