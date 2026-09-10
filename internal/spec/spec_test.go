package spec_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

func TestDefaultEnablesEveryFeature(t *testing.T) {
	s := spec.Default()

	if s.DefaultBranch != spec.DefaultBranch {
		t.Errorf("DefaultBranch = %q, want %q", s.DefaultBranch, spec.DefaultBranch)
	}
	if s.License != "MIT" {
		t.Errorf("License = %q, want MIT", s.License)
	}
	f := s.Features
	for name, enabled := range map[string]bool{
		"Git":             f.Git,
		"ClaudeConfig":    f.ClaudeConfig,
		"ClaudeAgents":    f.ClaudeAgents,
		"ClaudeWorkflows": f.ClaudeWorkflows,
		"GitHubActions":   f.GitHubActions,
		"Docs":            f.Docs,
		"Tests":           f.Tests,
		"PRTemplate":      f.PRTemplate,
		"CodingStandards": f.CodingStandards,
		"SecurityPolicy":  f.SecurityPolicy,
	} {
		if !enabled {
			t.Errorf("feature %s is disabled by default, want enabled", name)
		}
	}
}

func TestValidate(t *testing.T) {
	valid := func(mutate func(*spec.Spec)) spec.Spec {
		s := spec.Default()
		s.Name = "widget"
		s.Language = "go"
		if mutate != nil {
			mutate(&s)
		}
		return s
	}

	tests := []struct {
		name    string
		spec    spec.Spec
		wantErr string
	}{
		{name: "fully populated", spec: valid(nil)},
		{name: "dots and dashes allowed", spec: valid(func(s *spec.Spec) { s.Name = "acme.widget-2_0" })},
		{
			name:    "empty name",
			spec:    valid(func(s *spec.Spec) { s.Name = "" }),
			wantErr: "name: must not be empty",
		},
		{
			name:    "name starting with a dash",
			spec:    valid(func(s *spec.Spec) { s.Name = "-widget" }),
			wantErr: "must start with a letter or digit",
		},
		{
			name:    "name with a path separator",
			spec:    valid(func(s *spec.Spec) { s.Name = "acme/widget" }),
			wantErr: "must start with a letter or digit",
		},
		{
			name:    "over long name",
			spec:    valid(func(s *spec.Spec) { s.Name = strings.Repeat("a", 101) }),
			wantErr: "100 characters or fewer",
		},
		{
			name:    "missing language",
			spec:    valid(func(s *spec.Spec) { s.Language = "" }),
			wantErr: "language: must not be empty",
		},
		{
			name:    "empty branch",
			spec:    valid(func(s *spec.Spec) { s.DefaultBranch = "" }),
			wantErr: "default branch: must not be empty",
		},
		{
			name:    "branch with a space",
			spec:    valid(func(s *spec.Spec) { s.DefaultBranch = "my branch" }),
			wantErr: "Git does not allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("Validate() = %v, want nil", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("Validate() = nil, want error containing %q", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	s := spec.Spec{Name: "", Language: "", DefaultBranch: ""}

	err := s.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
	for _, want := range []string{"name:", "language:", "default branch:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() = %v, want it to mention %q", err, want)
		}
	}
}

func TestOptions(t *testing.T) {
	var s spec.Spec

	if got := s.Option("missing", "fallback"); got != "fallback" {
		t.Errorf("Option on a zero Spec = %q, want the fallback", got)
	}

	s.SetOption("go_module", "example.com/widget")
	s.SetOption("blank", "   ")

	if got := s.Option("go_module", "fallback"); got != "example.com/widget" {
		t.Errorf("Option = %q, want the stored value", got)
	}
	if got := s.Option("blank", "fallback"); got != "fallback" {
		t.Errorf("Option on a whitespace value = %q, want the fallback", got)
	}
}

func TestOptionKeysAreSorted(t *testing.T) {
	s := spec.Default()
	s.SetOption("zeta", "1")
	s.SetOption("alpha", "2")
	s.SetOption("mu", "3")

	got := s.OptionKeys()
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

func TestPathFallsBackToTheName(t *testing.T) {
	s := spec.Default()
	s.Name = "widget"

	got, err := s.Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("Path() = %q, want an absolute path", got)
	}
	if filepath.Base(got) != "widget" {
		t.Errorf("Path() = %q, want it to end in the repository name", got)
	}
}

func TestHasLicense(t *testing.T) {
	tests := map[string]bool{
		"MIT":                  true,
		"Apache-2.0":           true,
		spec.LicenseUnlicensed: false,
		"NONE":                 false,
		"":                     false,
	}
	for license, want := range tests {
		s := spec.Spec{License: license}
		if got := s.HasLicense(); got != want {
			t.Errorf("HasLicense(%q) = %v, want %v", license, got, want)
		}
	}
}
