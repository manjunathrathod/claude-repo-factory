package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*config.ProjectConfig)
		wantKind error
		wantMsg  string
	}{
		{name: "a complete configuration", mutate: nil},
		{
			name: "a language with several package managers",
			mutate: func(c *config.ProjectConfig) {
				c.Language = "python"
				c.PackageManager = "pip"
				c.ProjectType = config.ProjectTypeWorker
			},
		},
		{
			name:   "an empty output directory defaults to the project name",
			mutate: func(c *config.ProjectConfig) { c.OutputDirectory = "" },
		},

		// Project name.
		{
			name:     "empty project name",
			mutate:   func(c *config.ProjectConfig) { c.ProjectName = "" },
			wantKind: config.ErrMissingField,
		},
		{
			name:     "project name with a separator",
			mutate:   func(c *config.ProjectConfig) { c.ProjectName = "acme/widget" },
			wantKind: config.ErrUnsafeName,
		},
		{
			name:     "project name that is a device name",
			mutate:   func(c *config.ProjectConfig) { c.ProjectName = "nul" },
			wantKind: config.ErrUnsafeName,
		},

		// Output directory.
		{
			name:     "output directory escaping upwards",
			mutate:   func(c *config.ProjectConfig) { c.OutputDirectory = "../../etc" },
			wantKind: config.ErrPathTraversal,
		},
		{
			name:     "output directory at the filesystem root",
			mutate:   func(c *config.ProjectConfig) { c.OutputDirectory = "/" },
			wantKind: config.ErrInvalidValue,
		},

		// Branch.
		{
			name:     "empty branch",
			mutate:   func(c *config.ProjectConfig) { c.DefaultBranch = "" },
			wantKind: config.ErrMissingField,
		},
		{
			name:     "branch with a space",
			mutate:   func(c *config.ProjectConfig) { c.DefaultBranch = "my branch" },
			wantKind: config.ErrInvalidValue,
		},
		{
			name:     "branch with a double dot",
			mutate:   func(c *config.ProjectConfig) { c.DefaultBranch = "feature..x" },
			wantKind: config.ErrInvalidValue,
		},
		{
			name:     "branch beginning with a dash",
			mutate:   func(c *config.ProjectConfig) { c.DefaultBranch = "-main" },
			wantKind: config.ErrInvalidValue,
		},

		// Language.
		{
			name:     "empty language",
			mutate:   func(c *config.ProjectConfig) { c.Language = "" },
			wantKind: config.ErrMissingField,
			wantMsg:  "go, python",
		},
		{
			name:     "unregistered language",
			mutate:   func(c *config.ProjectConfig) { c.Language = "cobol" },
			wantKind: config.ErrUnsupportedLanguage,
			wantMsg:  "go, python",
		},
		{
			name:     "an alias is not a canonical language",
			mutate:   func(c *config.ProjectConfig) { c.Language = "golang" },
			wantKind: config.ErrUnsupportedLanguage,
		},

		// Project type.
		{
			name:     "empty project type",
			mutate:   func(c *config.ProjectConfig) { c.ProjectType = "" },
			wantKind: config.ErrMissingField,
		},
		{
			name:     "project type outside the vocabulary",
			mutate:   func(c *config.ProjectConfig) { c.ProjectType = "mainframe" },
			wantKind: config.ErrUnsupportedProjectType,
			wantMsg:  "api, cli, library, worker",
		},
		{
			name:     "project type the language does not offer",
			mutate:   func(c *config.ProjectConfig) { c.ProjectType = config.ProjectTypeWorker },
			wantKind: config.ErrUnsupportedProjectType,
			wantMsg:  "go supports",
		},

		// Package manager.
		{
			name:     "empty package manager",
			mutate:   func(c *config.ProjectConfig) { c.PackageManager = "" },
			wantKind: config.ErrMissingField,
		},
		{
			name:     "package manager from another ecosystem",
			mutate:   func(c *config.ProjectConfig) { c.PackageManager = "npm" },
			wantKind: config.ErrUnsupportedPackageManager,
			wantMsg:  "go supports gomod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validConfig()
			if tt.mutate != nil {
				tt.mutate(&c)
			}

			err := c.Validate(newFakeCatalog())

			if tt.wantKind == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want %v", tt.wantKind)
			}
			if !errors.Is(err, tt.wantKind) {
				t.Fatalf("Validate() = %v, want it to wrap %v", err, tt.wantKind)
			}
			if tt.wantMsg != "" && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("Validate() = %v, want the message to contain %q", err, tt.wantMsg)
			}
		})
	}
}

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	c := validConfig()
	c.ProjectName = "acme/widget"
	c.OutputDirectory = "../escape"
	c.DefaultBranch = "bad branch"
	c.Language = "cobol"

	err := c.Validate(newFakeCatalog())
	if err == nil {
		t.Fatal("Validate() = nil, want an error")
	}

	// A user should be able to fix one round of mistakes, not rediscover them
	// one command at a time.
	for _, want := range []string{"ProjectName", "OutputDirectory", "DefaultBranch", "Language"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() = %v, want it to mention %q", err, want)
		}
	}
}

func TestValidateStopsAfterAnUnusableLanguage(t *testing.T) {
	// With no valid language there is nothing to check a project type or a
	// package manager against, so reporting those too would be noise.
	c := validConfig()
	c.Language = "cobol"
	c.ProjectType = "mainframe"
	c.PackageManager = "cpan"

	err := c.Validate(newFakeCatalog())
	if err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
	if strings.Contains(err.Error(), "ProjectType") || strings.Contains(err.Error(), "PackageManager") {
		t.Errorf("Validate() = %v, want only the language failure reported", err)
	}
}

func TestValidateRequiresACatalog(t *testing.T) {
	// A nil catalog must be an error rather than a skipped check: silently
	// accepting any language would turn validation into a no-op.
	c := validConfig()

	err := c.Validate(nil)
	if err == nil {
		t.Fatal("Validate(nil) = nil, want an error")
	}
	if !errors.Is(err, config.ErrInvalidValue) {
		t.Fatalf("Validate(nil) = %v, want ErrInvalidValue", err)
	}
	if !strings.Contains(err.Error(), "catalog") {
		t.Errorf("Validate(nil) = %v, want it to explain that a catalog is required", err)
	}
}

func TestValidateStillChecksSafetyWithoutACatalog(t *testing.T) {
	// The catalog-free rules must not be skipped just because the catalog is
	// missing, or a nil catalog would hide an unsafe name.
	c := validConfig()
	c.ProjectName = "../etc"

	err := c.Validate(nil)
	if !errors.Is(err, config.ErrUnsafeName) {
		t.Fatalf("Validate(nil) = %v, want the unsafe name still reported", err)
	}
}

func TestFieldErrorCarriesTheField(t *testing.T) {
	c := validConfig()
	c.ProjectName = "acme/widget"

	err := c.Validate(newFakeCatalog())

	var fieldErr *config.FieldError
	if !errors.As(err, &fieldErr) {
		t.Fatalf("Validate() = %v, want a *config.FieldError in the chain", err)
	}
	if fieldErr.Field != "ProjectName" {
		t.Errorf("Field = %q, want ProjectName", fieldErr.Field)
	}
	if fieldErr.Value != "acme/widget" {
		t.Errorf("Value = %q, want the rejected value", fieldErr.Value)
	}
	if !strings.Contains(fieldErr.Error(), "ProjectName") {
		t.Errorf("Error() = %q, want it to name the field", fieldErr.Error())
	}
}
