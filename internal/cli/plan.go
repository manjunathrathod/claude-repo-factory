package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// universalArtifacts are the files every generated repository receives,
// whatever the language. Language plugins contribute additional files on top
// of these; the list lives here because it is a property of the factory, not
// of any one language.
var universalArtifacts = []string{
	"README.md",
	"CLAUDE.md",
	".gitignore",
	".claude/settings.json",
	".claude/agents/",
	".claude/commands/",
	".github/workflows/ci.yml",
	".github/PULL_REQUEST_TEMPLATE.md",
	"docs/architecture.md",
	"docs/coding-standards.md",
	"docs/security.md",
	"docs/testing.md",
	"tests/",
}

// writePlan prints the resolved specification and what generation would
// produce from it. It is the whole user-visible output of the current
// milestone, so it is kept deterministic and easy to assert on in tests.
func writePlan(w io.Writer, s spec.Spec, language plugin.Language) error {
	desc := language.Descriptor()
	path, err := s.Path()
	if err != nil {
		return fmt.Errorf("resolve target directory: %w", err)
	}

	fmt.Fprintf(w, "Repository plan for %q\n\n", s.Name)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	row := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			v = "-"
		}
		fmt.Fprintf(tw, "  %s\t%s\n", k, v)
	}
	row("Directory", path)
	row("Description", s.Description)
	row("Language", fmt.Sprintf("%s (%s)", desc.DisplayName, desc.ID))
	row("Project type", s.ProjectType)
	row("Author", s.Author)
	row("License", s.License)
	row("Initial branch", s.DefaultBranch)
	row("Remote", s.Remote)
	for _, key := range s.OptionKeys() {
		row(key, s.Options[key])
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(w, "\nFeatures\n")
	for _, f := range featureRows(s.Features) {
		fmt.Fprintf(w, "  %s %s\n", mark(f.enabled), f.name)
	}

	fmt.Fprintf(w, "\nArtifacts\n")
	for _, a := range universalArtifacts {
		fmt.Fprintf(w, "  %s\n", a)
	}
	fmt.Fprintf(w, "  (plus %s specific configuration)\n", desc.DisplayName)

	if cmds := language.Instructions(s).Commands; len(cmds) > 0 {
		fmt.Fprintf(w, "\nCommands recorded in CLAUDE.md and CI\n")
		ctw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, c := range cmds {
			fmt.Fprintf(ctw, "  %s\t%s\t%s\n", c.Name, c.Run, c.Description)
		}
		if err := ctw.Flush(); err != nil {
			return err
		}
	}
	return nil
}

type featureRow struct {
	name    string
	enabled bool
}

func featureRows(f spec.Features) []featureRow {
	return []featureRow{
		{"Git initialisation", f.Git},
		{"Claude configuration (.claude)", f.ClaudeConfig},
		{"Claude specialist agents", f.ClaudeAgents},
		{"Claude feature/review/fix workflows", f.ClaudeWorkflows},
		{"GitHub Actions CI", f.GitHubActions},
		{"Documentation structure", f.Docs},
		{"Test structure", f.Tests},
		{"Pull request template", f.PRTemplate},
		{"Coding standards", f.CodingStandards},
		{"Security instructions", f.SecurityPolicy},
	}
}

func mark(enabled bool) string {
	if enabled {
		return "[x]"
	}
	return "[ ]"
}
