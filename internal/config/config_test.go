package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
)

func TestDefaultEnablesEveryFeature(t *testing.T) {
	c := config.Default()

	if c.DefaultBranch != config.DefaultBranch {
		t.Errorf("DefaultBranch = %q, want %q", c.DefaultBranch, config.DefaultBranch)
	}
	if c.License != "MIT" {
		t.Errorf("License = %q, want MIT", c.License)
	}

	// "Professional by default" means the cheapest path produces the most
	// complete repository; flags only ever turn things off.
	toggles := map[string]bool{
		"InitializeGit":          c.InitializeGit,
		"IncludeClaude":          c.IncludeClaude,
		"IncludeClaudeAgents":    c.IncludeClaudeAgents,
		"IncludeClaudeWorkflows": c.IncludeClaudeWorkflows,
		"IncludeGitHubActions":   c.IncludeGitHubActions,
		"IncludeDocs":            c.IncludeDocs,
		"IncludeTests":           c.IncludeTests,
		"IncludePRTemplate":      c.IncludePRTemplate,
		"IncludeCodingStandards": c.IncludeCodingStandards,
		"IncludeSecurityPolicy":  c.IncludeSecurityPolicy,
	}
	for name, enabled := range toggles {
		if !enabled {
			t.Errorf("%s is disabled by default, want enabled", name)
		}
	}
}

func TestDefaultIsNotSharedBetweenCallers(t *testing.T) {
	// Default returns a value, but Options is a reference type: two callers
	// must not end up writing into the same map.
	a := config.Default()
	b := config.Default()

	a.SetOption("go_module", "example.com/a")

	if got := b.Option("go_module", ""); got != "" {
		t.Fatalf("mutating one Default leaked into another: got %q", got)
	}
}

func TestOptions(t *testing.T) {
	var c config.ProjectConfig

	if got := c.Option("missing", "fallback"); got != "fallback" {
		t.Errorf("Option on a zero value = %q, want the fallback", got)
	}

	c.SetOption("go_module", "example.com/widget")
	c.SetOption("blank", "   ")

	if got := c.Option("go_module", "fallback"); got != "example.com/widget" {
		t.Errorf("Option = %q, want the stored value", got)
	}
	if got := c.Option("blank", "fallback"); got != "fallback" {
		t.Errorf("Option on a whitespace value = %q, want the fallback", got)
	}
}

func TestOptionKeysAreSorted(t *testing.T) {
	c := config.Default()
	c.SetOption("zeta", "1")
	c.SetOption("alpha", "2")
	c.SetOption("mu", "3")

	got := c.OptionKeys()
	want := []string{"alpha", "mu", "zeta"}

	if len(got) != len(want) {
		t.Fatalf("OptionKeys() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("OptionKeys() = %v, want %v", got, want)
		}
	}
}

// ResolvedOutputDirectory is the parent; the repository is created inside it.
// An empty OutputDirectory means the working directory, so both methods still
// agree on <cwd>/<project> in the common case.
func TestResolvedOutputDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tests := []struct {
		name     string
		project  string
		output   string
		wantBase string
	}{
		{name: "empty output is the working directory", project: "widget", output: "", wantBase: filepath.Base(cwd)},
		{name: "whitespace output is the working directory", project: "widget", output: "   ", wantBase: filepath.Base(cwd)},
		{name: "explicit output is used as the parent", project: "widget", output: "elsewhere", wantBase: "elsewhere"},
		{name: "nested output", project: "widget", output: "projects/team", wantBase: "team"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.Default()
			c.ProjectName = tt.project
			c.OutputDirectory = tt.output

			got, err := c.ResolvedOutputDirectory()
			if err != nil {
				t.Fatalf("ResolvedOutputDirectory() error = %v", err)
			}
			if !filepath.IsAbs(got) {
				t.Errorf("ResolvedOutputDirectory() = %q, want an absolute path", got)
			}
			if filepath.Base(got) != tt.wantBase {
				t.Errorf("ResolvedOutputDirectory() = %q, want it to end in %q", got, tt.wantBase)
			}
		})
	}
}

func TestResolvedProjectDirectory(t *testing.T) {
	tests := []struct {
		name    string
		project string
		output  string
	}{
		{name: "explicit parent", project: "payment-api", output: "projects"},
		{name: "nested parent", project: "payment-api", output: "projects/team"},
		{name: "no parent", project: "payment-api", output: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := config.Default()
			c.ProjectName = tt.project
			c.OutputDirectory = tt.output

			base, err := c.ResolvedOutputDirectory()
			if err != nil {
				t.Fatalf("ResolvedOutputDirectory() error = %v", err)
			}
			got, err := c.ResolvedProjectDirectory()
			if err != nil {
				t.Fatalf("ResolvedProjectDirectory() error = %v", err)
			}

			if want := filepath.Join(base, tt.project); got != want {
				t.Errorf("ResolvedProjectDirectory() = %q, want %q", got, want)
			}
			if filepath.Base(got) != tt.project {
				t.Errorf("ResolvedProjectDirectory() = %q, want it to end in the project name", got)
			}
			if filepath.Dir(got) != base {
				t.Errorf("parent of %q = %q, want %q", got, filepath.Dir(got), base)
			}
		})
	}
}

func TestHasLicense(t *testing.T) {
	tests := map[string]bool{
		"MIT":                    true,
		"Apache-2.0":             true,
		config.LicenseUnlicensed: false,
		"NONE":                   false,
		"":                       false,
		"   ":                    false,
	}

	for license, want := range tests {
		c := config.ProjectConfig{License: license}
		if got := c.HasLicense(); got != want {
			t.Errorf("HasLicense(%q) = %v, want %v", license, got, want)
		}
	}
}

func TestProjectTypeVocabulary(t *testing.T) {
	want := []config.ProjectType{
		config.ProjectTypeAPI,
		config.ProjectTypeCLI,
		config.ProjectTypeLibrary,
		config.ProjectTypeWorker,
	}

	got := config.ProjectTypes()
	if config.JoinProjectTypes(got, ",") != config.JoinProjectTypes(want, ",") {
		t.Fatalf("ProjectTypes() = %v, want %v", got, want)
	}
	for _, p := range want {
		if !p.Valid() {
			t.Errorf("%q.Valid() = false, want true", p)
		}
	}
	if config.ProjectType("service").Valid() {
		t.Error("service is still valid; it should have been remapped to api")
	}
	if config.ProjectType("").Valid() {
		t.Error("the empty project type reports as valid")
	}
}

func TestProjectTypesCannotBeMutatedByCallers(t *testing.T) {
	first := config.ProjectTypes()
	first[0] = "tampered"

	if config.ProjectTypes()[0] != config.ProjectTypeAPI {
		t.Fatal("ProjectTypes() exposed the package-level vocabulary to mutation")
	}
}

func TestParseProjectType(t *testing.T) {
	tests := []struct {
		input   string
		want    config.ProjectType
		wantErr bool
	}{
		{input: "api", want: config.ProjectTypeAPI},
		{input: "CLI", want: config.ProjectTypeCLI},
		{input: "  library  ", want: config.ProjectTypeLibrary},
		{input: "Worker", want: config.ProjectTypeWorker},
		{input: "service", wantErr: true},
		{input: "", wantErr: true},
		{input: "mainframe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := config.ParseProjectType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseProjectType(%q) = %q, want an error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseProjectType(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseProjectType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseNormalisesLanguageAndPackageManager(t *testing.T) {
	if got := config.ParseLanguage("  NodeJS "); got != "nodejs" {
		t.Errorf("ParseLanguage = %q, want nodejs", got)
	}
	if got := config.ParsePackageManager(" NPM "); got != "npm" {
		t.Errorf("ParsePackageManager = %q, want npm", got)
	}
}
