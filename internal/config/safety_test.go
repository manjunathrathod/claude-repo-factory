package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKind error
	}{
		// Accepted.
		{name: "simple lowercase", input: "widget"},
		{name: "digits and dashes", input: "widget-2"},
		{name: "underscores", input: "my_widget"},
		{name: "dots inside", input: "acme.widget"},
		{name: "mixed case", input: "MyWidget"},
		{name: "starts with a digit", input: "2fa-service"},
		{name: "at the length limit", input: strings.Repeat("a", config.MaxProjectNameLength)},
		{name: "reserved word as a prefix only", input: "console"},
		{name: "reserved word as a suffix only", input: "my-con"},
		// COM0 and LPT0 are not in the documented reserved set.
		{name: "com0 is not reserved", input: "com0"},
		{name: "lpt0 is not reserved", input: "lpt0"},

		// Missing.
		{name: "empty", input: "", wantKind: config.ErrMissingField},
		{name: "only whitespace", input: "   ", wantKind: config.ErrMissingField},

		// Unsafe.
		{name: "leading space", input: " widget", wantKind: config.ErrUnsafeName},
		{name: "trailing space", input: "widget ", wantKind: config.ErrUnsafeName},
		{name: "one over the length limit", input: strings.Repeat("a", config.MaxProjectNameLength+1), wantKind: config.ErrUnsafeName},
		{name: "forward slash", input: "acme/widget", wantKind: config.ErrUnsafeName},
		{name: "backslash", input: `acme\widget`, wantKind: config.ErrUnsafeName},
		{name: "leading dash", input: "-widget", wantKind: config.ErrUnsafeName},
		{name: "leading dot", input: ".widget", wantKind: config.ErrUnsafeName},
		{name: "single dot", input: ".", wantKind: config.ErrUnsafeName},
		{name: "double dot", input: "..", wantKind: config.ErrUnsafeName},
		{name: "traversal", input: "../etc", wantKind: config.ErrUnsafeName},
		{name: "embedded double dot", input: "a..b", wantKind: config.ErrUnsafeName},
		{name: "trailing dot", input: "widget.", wantKind: config.ErrUnsafeName},
		{name: "inner space", input: "my widget", wantKind: config.ErrUnsafeName},
		{name: "shell metacharacter", input: "widget;rm", wantKind: config.ErrUnsafeName},
		{name: "shell expansion", input: "widget$(id)", wantKind: config.ErrUnsafeName},
		{name: "tilde", input: "~widget", wantKind: config.ErrUnsafeName},
		{name: "colon", input: "C:widget", wantKind: config.ErrUnsafeName},
		{name: "newline", input: "widget\nrm", wantKind: config.ErrUnsafeName},
		{name: "NUL byte", input: "widget\x00", wantKind: config.ErrUnsafeName},
		{name: "non-ascii", input: "wídget", wantKind: config.ErrUnsafeName},

		// Windows reserved device names.
		{name: "reserved CON", input: "CON", wantKind: config.ErrUnsafeName},
		{name: "reserved lowercase nul", input: "nul", wantKind: config.ErrUnsafeName},
		{name: "reserved mixed case Aux", input: "Aux", wantKind: config.ErrUnsafeName},
		{name: "reserved with extension", input: "nul.txt", wantKind: config.ErrUnsafeName},
		{name: "reserved COM1", input: "COM1", wantKind: config.ErrUnsafeName},
		{name: "reserved LPT9", input: "LPT9", wantKind: config.ErrUnsafeName},
		{name: "reserved PRN", input: "prn", wantKind: config.ErrUnsafeName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateProjectName(tt.input)

			if tt.wantKind == nil {
				if err != nil {
					t.Fatalf("ValidateProjectName(%q) = %v, want nil", tt.input, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateProjectName(%q) = nil, want %v", tt.input, tt.wantKind)
			}
			if !errors.Is(err, tt.wantKind) {
				t.Fatalf("ValidateProjectName(%q) = %v, want it to wrap %v", tt.input, err, tt.wantKind)
			}

			var fieldErr *config.FieldError
			if !errors.As(err, &fieldErr) {
				t.Fatalf("ValidateProjectName(%q) = %v, want a *config.FieldError", tt.input, err)
			}
			if fieldErr.Field != "ProjectName" {
				t.Errorf("Field = %q, want ProjectName", fieldErr.Field)
			}
			if fieldErr.Reason == "" {
				t.Error("Reason is empty; the user would not know what to fix")
			}
		})
	}
}

func TestValidateOutputDirectory(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKind error
	}{
		// Accepted.
		{name: "empty means use the project name", input: ""},
		{name: "bare name", input: "widget"},
		{name: "nested relative", input: "projects/widget"},
		{name: "explicit current directory", input: "./widget"},
		{name: "posix absolute", input: "/home/dev/widget"},
		{name: "windows absolute", input: `C:\dev\widget`},
		{name: "windows absolute forward slashes", input: "C:/dev/widget"},
		{name: "dot segment that cleans away", input: "projects/./widget"},
		{name: "name containing dots", input: "projects/v1.2/widget"},

		// Traversal.
		{name: "leading parent", input: "../widget", wantKind: config.ErrPathTraversal},
		{name: "bare parent", input: "..", wantKind: config.ErrPathTraversal},
		{name: "embedded parent", input: "projects/../../etc/widget", wantKind: config.ErrPathTraversal},
		{name: "backslash traversal", input: `..\..\Windows\System32`, wantKind: config.ErrPathTraversal},
		{name: "traversal escaping an absolute path", input: "/srv/../../etc", wantKind: config.ErrPathTraversal},
		{name: "traversal that cleans to a parent", input: "a/b/../../../c", wantKind: config.ErrPathTraversal},
		// Strict policy: rejected even though it resolves back inside the base.
		{name: "inner parent that does not escape", input: "a/b/../c", wantKind: config.ErrPathTraversal},
		{name: "windows mixed separators traversal", input: `projects\..\..\etc`, wantKind: config.ErrPathTraversal},

		// Otherwise invalid.
		{name: "NUL byte", input: "widget\x00/etc", wantKind: config.ErrInvalidValue},
		{name: "posix root", input: "/", wantKind: config.ErrInvalidValue},
		{name: "windows drive root", input: `C:\`, wantKind: config.ErrInvalidValue},
		{name: "windows drive root bare", input: "C:", wantKind: config.ErrInvalidValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateOutputDirectory(tt.input)

			if tt.wantKind == nil {
				if err != nil {
					t.Fatalf("ValidateOutputDirectory(%q) = %v, want nil", tt.input, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateOutputDirectory(%q) = nil, want %v", tt.input, tt.wantKind)
			}
			if !errors.Is(err, tt.wantKind) {
				t.Fatalf("ValidateOutputDirectory(%q) = %v, want it to wrap %v", tt.input, err, tt.wantKind)
			}
		})
	}
}

// A name that passes validation must never be able to contribute more than a
// single path segment. This is the property the traversal defence rests on.
func TestValidProjectNameIsAlwaysASingleSegment(t *testing.T) {
	names := []string{"widget", "my-widget", "acme.widget", "a_b.c-d", "X1"}

	for _, name := range names {
		if err := config.ValidateProjectName(name); err != nil {
			t.Fatalf("ValidateProjectName(%q) = %v, want nil", name, err)
		}
		if strings.ContainsAny(name, `/\`) {
			t.Errorf("%q passed validation but contains a path separator", name)
		}
	}
}
