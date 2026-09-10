package cli_test

import (
	"strings"
	"testing"
)

// The summary block is the last thing a user reads before committing to a
// configuration, so its exact shape is asserted rather than sampled.
func TestSummaryBlockShape(t *testing.T) {
	out, _, err := run(t, nil,
		"create", "payment-api",
		"--language", "node",
		"--type", "api",
		"--package-manager", "npm",
		"--description", "Payment service",
		"--yes",
	)
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	lines := summaryBlock(t, out)
	if len(lines) < 11 {
		t.Fatalf("summary has %d lines, want a title, a rule and nine rows\n%s", len(lines), out)
	}

	if lines[0] != "Project Configuration" {
		t.Errorf("title = %q", lines[0])
	}
	if lines[1] != strings.Repeat("-", len("Project Configuration")) {
		t.Errorf("rule = %q, want it to match the title length", lines[1])
	}

	wantRows := []string{
		"Name: payment-api",
		"Description: Payment service",
		"Language: Node.js / TypeScript",
		"Type: API",
		"Package Manager: npm",
	}
	for i, want := range wantRows {
		if lines[2+i] != want {
			t.Errorf("row %d = %q, want %q", i, lines[2+i], want)
		}
	}
}

func TestSummaryRendersBooleansAsYesNo(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
		want  []string
	}{
		{
			name:  "everything on",
			flags: nil,
			want:  []string{"Initialize Git: Yes", "Claude Code Setup: Yes", "GitHub Actions: Yes"},
		},
		{
			name:  "everything off",
			flags: []string{"--no-git", "--no-claude", "--no-ci"},
			want:  []string{"Initialize Git: No", "Claude Code Setup: No", "GitHub Actions: No"},
		},
		{
			name:  "mixed",
			flags: []string{"--no-git", "--no-ci"},
			want:  []string{"Initialize Git: No", "Claude Code Setup: Yes", "GitHub Actions: No"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"create", "widget", "-l", "go", "--yes"}, tt.flags...)
			out, _, err := run(t, nil, args...)
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("summary is missing %q\n%s", want, out)
				}
			}
		})
	}
}

func TestSummaryShowsAPlaceholderForAnEmptyValue(t *testing.T) {
	// A description is optional; the row must still be present so the block
	// always has the same shape.
	out, _, err := run(t, nil, "create", "widget", "-l", "go", "--description", " ", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "Description:") {
		t.Errorf("summary dropped the description row\n%s", out)
	}
}

func TestSummaryShowsAnAbsoluteOutputDirectory(t *testing.T) {
	out, _, err := run(t, nil, "create", "widget", "-l", "go", "--dir", "services", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	dir := summaryValue(t, out, "Output Directory")
	if dir == "services" {
		t.Errorf("output directory = %q, want it resolved to an absolute path", dir)
	}
	if !strings.HasSuffix(dir, "services") {
		t.Errorf("output directory = %q, want it to end in the requested directory", dir)
	}
}

func TestSummaryUsesThePluginDisplayNames(t *testing.T) {
	// The display text belongs to the plugin, so the CLI must not invent its
	// own casing or wording for a language or a project type.
	tests := []struct {
		language string
		typ      string
		want     []string
	}{
		{language: "go", typ: "worker", want: []string{"Language: Go", "Type: Worker"}},
		{language: "python", typ: "api", want: []string{"Language: Python", "Type: API"}},
		{language: "java", typ: "library", want: []string{"Language: Java", "Type: Library"}},
		{language: "nodejs", typ: "cli", want: []string{"Language: Node.js / TypeScript", "Type: CLI"}},
	}

	for _, tt := range tests {
		t.Run(tt.language+"/"+tt.typ, func(t *testing.T) {
			out, _, err := run(t, nil, "create", "widget", "-l", tt.language, "-t", tt.typ, "--yes")
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("summary is missing %q\n%s", want, out)
				}
			}
		})
	}
}

// summaryBlock returns the summary lines, starting at the title.
func summaryBlock(t *testing.T, out string) []string {
	t.Helper()

	lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if line == "Project Configuration" {
			return lines[i:]
		}
	}
	t.Fatalf("output has no summary block\n%s", out)
	return nil
}
