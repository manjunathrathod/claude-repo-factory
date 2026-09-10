package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
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
func writePlan(w io.Writer, cfg config.ProjectConfig, language plugin.Language) error {
	desc := language.Descriptor()
	path, err := cfg.ResolvedOutputDirectory()
	if err != nil {
		return fmt.Errorf("resolve target directory: %w", err)
	}

	fmt.Fprintf(w, "Repository plan for %q\n\n", cfg.ProjectName)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	row := func(k, v string) {
		if strings.TrimSpace(v) == "" {
			v = "-"
		}
		fmt.Fprintf(tw, "  %s\t%s\n", k, v)
	}
	row("Directory", path)
	row("Description", cfg.Description)
	row("Language", fmt.Sprintf("%s (%s)", desc.DisplayName, desc.ID))
	row("Project type", string(cfg.ProjectType))
	row("Package manager", string(cfg.PackageManager))
	row("Author", cfg.Author)
	row("License", cfg.License)
	row("Initial branch", cfg.DefaultBranch)
	row("Remote", cfg.Remote)
	for _, key := range cfg.OptionKeys() {
		row(key, cfg.Options[key])
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintf(w, "\nFeatures\n")
	for _, f := range featureRows(cfg) {
		fmt.Fprintf(w, "  %s %s\n", mark(f.enabled), f.name)
	}

	fmt.Fprintf(w, "\nArtifacts\n")
	for _, a := range universalArtifacts {
		fmt.Fprintf(w, "  %s\n", a)
	}
	fmt.Fprintf(w, "  (plus %s specific configuration)\n", desc.DisplayName)

	if cmds := language.Instructions(cfg).Commands; len(cmds) > 0 {
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

func featureRows(c config.ProjectConfig) []featureRow {
	return []featureRow{
		{"Git initialisation", c.InitializeGit},
		{"Claude configuration (.claude)", c.IncludeClaude},
		{"Claude specialist agents", c.IncludeClaudeAgents},
		{"Claude feature/review/fix workflows", c.IncludeClaudeWorkflows},
		{"GitHub Actions CI", c.IncludeGitHubActions},
		{"Documentation structure", c.IncludeDocs},
		{"Test structure", c.IncludeTests},
		{"Pull request template", c.IncludePRTemplate},
		{"Coding standards", c.IncludeCodingStandards},
		{"Security instructions", c.IncludeSecurityPolicy},
	}
}

func mark(enabled bool) string {
	if enabled {
		return "[x]"
	}
	return "[ ]"
}
