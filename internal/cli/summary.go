package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
)

// summaryTitle heads the confirmation block. The rule beneath it is derived
// from its length rather than hard-coded, so the two can never disagree.
const summaryTitle = "Project Configuration"

// summaryRow is one label/value line of the confirmation block.
type summaryRow struct {
	Label string
	Value string
}

// summaryRows renders the resolved configuration as the lines the user is
// asked to confirm.
//
// It is separate from writeSummary, and returns data rather than text, so a
// test can assert on the values a user is shown without parsing formatted
// output. The order is fixed: it is the order the questions were asked in.
func summaryRows(cfg config.ProjectConfig, desc plugin.Descriptor, dir string) []summaryRow {
	return []summaryRow{
		{"Name", cfg.ProjectName},
		{"Description", cfg.Description},
		{"Language", desc.DisplayName},
		{"Type", desc.ProjectTypeDisplayName(cfg.ProjectType)},
		{"Package Manager", string(cfg.PackageManager)},
		{"Output Directory", dir},
		{"Initialize Git", yesNo(cfg.InitializeGit)},
		{"Claude Code Setup", yesNo(cfg.IncludeClaude)},
		{"GitHub Actions", yesNo(cfg.IncludeGitHubActions)},
	}
}

// writeSummary prints the configuration block shown before the final
// confirmation. The output is plain and deterministic: it is both the last
// thing a user reads before committing and the thing tests assert on.
func writeSummary(w io.Writer, cfg config.ProjectConfig, desc plugin.Descriptor) error {
	// Resolved here rather than at prompt time so the user confirms the
	// absolute path that would actually be written to, not what they typed.
	dir, err := cfg.ResolvedOutputDirectory()
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}

	fmt.Fprintln(w, summaryTitle)
	fmt.Fprintln(w, strings.Repeat("-", len(summaryTitle)))
	for _, row := range summaryRows(cfg, desc, dir) {
		fmt.Fprintf(w, "%s: %s\n", row.Label, displayValue(row.Value))
	}
	return nil
}

// displayValue renders one value as a single line.
//
// Validation already rejects control characters, so this is defence in depth:
// the summary is the gate the user reads before agreeing, and a value that
// could forge a row would let it lie about what is about to happen. Belt and
// braces is warranted for the one block whose integrity the confirmation
// depends on.
func displayValue(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	if strings.ContainsFunc(v, unicode.IsControl) {
		return strconv.QuoteToASCII(v)
	}
	return v
}

// yesNo renders a boolean the way the summary and the prompts present it.
func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
