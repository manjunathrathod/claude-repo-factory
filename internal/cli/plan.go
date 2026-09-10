package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
)

// universalArtifact is a file every generated repository receives, whatever
// the language, together with the feature that decides whether it is written.
// Language plugins contribute additional files on top of these; the list
// lives here because it is a property of the factory, not of any one
// language.
type universalArtifact struct {
	path string
	// enabled reports whether this configuration produces the file. A nil
	// enabled means the file is unconditional.
	enabled func(config.ProjectConfig) bool
}

var universalArtifacts = []universalArtifact{
	{path: "README.md"},
	{path: "CLAUDE.md", enabled: func(c config.ProjectConfig) bool { return c.IncludeClaude }},
	{path: ".gitignore"},
	{path: ".claude/settings.json", enabled: func(c config.ProjectConfig) bool { return c.IncludeClaude }},
	{path: ".claude/agents/", enabled: func(c config.ProjectConfig) bool { return c.IncludeClaudeAgents }},
	{path: ".claude/commands/", enabled: func(c config.ProjectConfig) bool { return c.IncludeClaudeWorkflows }},
	{path: ".github/workflows/ci.yml", enabled: func(c config.ProjectConfig) bool { return c.IncludeGitHubActions }},
	{path: ".github/PULL_REQUEST_TEMPLATE.md", enabled: func(c config.ProjectConfig) bool { return c.IncludePRTemplate }},
	{path: "docs/architecture.md", enabled: func(c config.ProjectConfig) bool { return c.IncludeDocs }},
	{path: "docs/coding-standards.md", enabled: func(c config.ProjectConfig) bool { return c.IncludeCodingStandards }},
	{path: "docs/security.md", enabled: func(c config.ProjectConfig) bool { return c.IncludeSecurityPolicy }},
	{path: "docs/testing.md", enabled: func(c config.ProjectConfig) bool { return c.IncludeDocs }},
	{path: "tests/", enabled: func(c config.ProjectConfig) bool { return c.IncludeTests }},
}

// artifactsFor returns the universal files this configuration would produce.
// The plan must never promise a file the feature list above it says is off.
func artifactsFor(c config.ProjectConfig) []string {
	paths := make([]string, 0, len(universalArtifacts))
	for _, a := range universalArtifacts {
		if a.enabled == nil || a.enabled(c) {
			paths = append(paths, a.path)
		}
	}
	return paths
}

// writePlan prints the resolved specification and what generation would
// produce from it. It is the whole user-visible output of the current
// milestone, so it is kept deterministic and easy to assert on in tests.
func writePlan(w io.Writer, cfg config.ProjectConfig, language plugin.Language) error {
	desc := language.Descriptor()
	// The repository directory, not its parent: this row answers "where will
	// this end up", which the summary's Output Directory deliberately does not.
	path, err := cfg.ResolvedProjectDirectory()
	if err != nil {
		return fmt.Errorf("resolve target directory: %w", err)
	}

	fmt.Fprintf(w, "Repository plan for %q\n\n", cfg.ProjectName)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	row := func(k, v string) {
		fmt.Fprintf(tw, "  %s\t%s\n", k, displayValue(v))
	}
	row("Directory", path)
	row("Description", cfg.Description)
	row("Language", fmt.Sprintf("%s (%s)", desc.DisplayName, desc.ID))
	row("Project type", desc.ProjectTypeDisplayName(cfg.ProjectType))
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
	for _, a := range artifactsFor(cfg) {
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
