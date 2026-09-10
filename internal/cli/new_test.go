package cli_test

import (
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/prompt"
)

func TestNewWithFlagsPrintsThePlan(t *testing.T) {
	out, _, err := run(t, nil,
		"new", "widget",
		"--language", "go",
		"--type", "api",
		"--description", "Widget control plane",
		"--author", "Platform Team",
		"--license", "Apache-2.0",
		"--set", "go_module=github.com/acme/widget",
		"--yes",
	)
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}

	for _, want := range []string{
		"widget",
		"Widget control plane",
		"Go (go)",
		"api",
		"Platform Team",
		"Apache-2.0",
		"github.com/acme/widget",
		"CLAUDE.md",
		".claude/agents/",
		".github/workflows/ci.yml",
		"go test ./...",
		"not implemented yet",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plan is missing %q\n%s", want, out)
		}
	}
}

func TestNewResolvesPackageManager(t *testing.T) {
	out, _, err := run(t, nil, "new", "widget", "-l", "python", "--package-manager", "poetry", "--yes")
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "poetry") {
		t.Errorf("plan is missing the chosen package manager\n%s", out)
	}
}

func TestNewDefaultsThePackageManager(t *testing.T) {
	out, _, err := run(t, nil, "new", "widget", "-l", "python", "--yes")
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}
	// uv is the declared default for python; a single-manager language such
	// as go must resolve without asking at all.
	if !strings.Contains(out, "Package manager  uv") {
		t.Errorf("plan did not use the declared default package manager\n%s", out)
	}
}

func TestNewDoesNotPromptWhenALanguageHasOnePackageManager(t *testing.T) {
	asker := prompt.NewScripted(map[string]string{
		"Project type":   "cli",
		"Go module path": "github.com/acme/widget",
		"Author":         "Platform Team",
		"License":        "MIT",
	})
	asker.Answers["One line description"] = "Widget"

	_, _, err := run(t, asker, "new", "widget", "-l", "go")
	if err != nil {
		t.Fatalf("new error = %v", err)
	}
	if containsString(asker.Asked, "Package manager") {
		t.Error("the user was asked to choose between one package manager")
	}
}

func TestNewResolvesAnAlias(t *testing.T) {
	out, _, err := run(t, nil, "new", "widget", "-l", "typescript", "--yes")
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "Node.js / TypeScript (nodejs)") {
		t.Errorf("plan did not resolve the alias\n%s", out)
	}
}

func TestNewUsesTheLanguageDefaultProjectType(t *testing.T) {
	out, _, err := run(t, nil, "new", "widget", "-l", "python", "--yes")
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "Project type     library") {
		t.Errorf("plan did not use the default project type\n%s", out)
	}
}

func TestNewAppliesLanguageOptionDefaults(t *testing.T) {
	out, _, err := run(t, nil, "new", "my-widget", "-l", "python", "--yes")
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "my_widget") {
		t.Errorf("plan is missing the derived package name\n%s", out)
	}
}

func TestNewSkipFlagsTurnFeaturesOff(t *testing.T) {
	out, _, err := run(t, nil, "new", "widget", "-l", "go", "--yes", "--no-ci", "--no-git", "--no-docs", "--no-claude-workflows")
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}

	for _, want := range []string{
		"[ ] Git initialisation",
		"[ ] GitHub Actions CI",
		"[ ] Documentation structure",
		"[ ] Claude feature/review/fix workflows",
		"[x] Claude specialist agents",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("feature list is missing %q\n%s", want, out)
		}
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "unknown language",
			args:    []string{"new", "widget", "-l", "cobol", "--yes"},
			wantErr: "unknown language",
		},
		{
			name:    "planned language",
			args:    []string{"new", "widget", "-l", "rust", "--yes"},
			wantErr: "not available yet",
		},
		{
			name:    "unknown project type",
			args:    []string{"new", "widget", "-l", "go", "-t", "mainframe", "--yes"},
			wantErr: "must be one of api, cli, library, worker",
		},
		{
			name:    "invalid repository name",
			args:    []string{"new", "acme/widget", "-l", "go", "--yes"},
			wantErr: "ProjectName:",
		},
		{
			name:    "project type the language does not offer",
			args:    []string{"new", "widget", "-l", "go", "-t", "api", "--package-manager", "npm", "--yes"},
			wantErr: "does not support package manager",
		},
		{
			name:    "malformed set option",
			args:    []string{"new", "widget", "-l", "go", "--set", "novalue", "--yes"},
			wantErr: "expected key=value",
		},
		{
			name:    "too many positional arguments",
			args:    []string{"new", "widget", "extra", "--yes"},
			wantErr: "accepts at most 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := run(t, nil, tt.args...)
			if err == nil {
				t.Fatalf("new = nil error, want one containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("new error = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestNewPromptsForEverythingItWasNotGiven(t *testing.T) {
	asker := prompt.NewScripted(map[string]string{
		"Repository name":      "widget",
		"One line description": "Widget control plane",
		"Primary language":     "go",
		"Project type":         "cli",
		"Author":               "Platform Team",
		"License":              "MIT",
		"Go module path":       "github.com/acme/widget",
	})

	out, _, err := run(t, asker, "new")
	if err != nil {
		t.Fatalf("new error = %v\n%s", err, out)
	}

	for _, want := range []string{"widget", "Widget control plane", "Go (go)", "cli", "github.com/acme/widget"} {
		if !strings.Contains(out, want) {
			t.Errorf("plan is missing %q\n%s", want, out)
		}
	}

	for _, wantPrompt := range []string{"Repository name", "Primary language", "Project type", "Go module path"} {
		if !containsString(asker.Asked, wantPrompt) {
			t.Errorf("the user was never asked %q (asked: %v)", wantPrompt, asker.Asked)
		}
	}
}

func TestNewDoesNotPromptForAnswersGivenAsFlags(t *testing.T) {
	asker := prompt.NewScripted(map[string]string{"Project type": "cli"})

	_, _, err := run(t, asker,
		"new", "widget",
		"-l", "go",
		"--description", "Widget control plane",
		"--author", "Platform Team",
		"--license", "MIT",
		"--set", "go_module=github.com/acme/widget",
	)
	if err != nil {
		t.Fatalf("new error = %v", err)
	}

	for _, unwanted := range []string{"Repository name", "Primary language", "Author", "License", "Go module path"} {
		if containsString(asker.Asked, unwanted) {
			t.Errorf("the user was asked %q even though it was passed as a flag", unwanted)
		}
	}
}

func TestNewPropagatesAPromptFailure(t *testing.T) {
	// An asker with no scripted answers stands in for a cancelled prompt.
	_, _, err := run(t, prompt.NewScripted(nil), "new")
	if err == nil {
		t.Fatal("new = nil error, want the prompt failure to surface")
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
