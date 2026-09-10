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

func TestValidateNoControlCharacters(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "plain text", input: "Payment service"},
		{name: "punctuation", input: "A service: fast, safe & tested (v2)"},
		{name: "non-ascii letters", input: "Zahlungsdienst für Kunden"},
		{name: "empty", input: ""},

		{name: "newline", input: "Widget\nInitialize Git: No", wantErr: true},
		{name: "carriage return", input: "Widget\rOverwritten", wantErr: true},
		{name: "tab", input: "Widget\tvalue", wantErr: true},
		{name: "ANSI escape", input: "Widget\x1b[2J", wantErr: true},
		{name: "NUL", input: "Widget\x00", wantErr: true},
		{name: "backspace", input: "Widget\b\b\b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateNoControlCharacters("Description", tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateNoControlCharacters(%q) = nil, want an error", tt.input)
				}
				// The rejected value must be escaped in the message, never
				// replayed, or the error output could be forged instead.
				if strings.Contains(err.Error(), "\n") {
					t.Errorf("error message replays a raw newline: %q", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateNoControlCharacters(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

func TestValidateRemote(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty means no remote", input: ""},
		{name: "https", input: "https://github.com/acme/widget.git"},
		{name: "ssh scheme", input: "ssh://git@github.com/acme/widget.git"},
		{name: "scp style", input: "git@github.com:acme/widget.git"},
		{name: "git scheme", input: "git://example.com/widget.git"},
		{name: "file scheme", input: "file:///srv/git/widget.git"},
		{name: "posix path", input: "/srv/git/widget.git"},
		{name: "relative path", input: "./mirror"},
		{name: "windows path", input: `C:\git\widget`},

		{name: "leading dash reads as a git flag", input: "--upload-pack=calc", wantErr: true},
		{name: "single dash", input: "-o", wantErr: true},
		{name: "ext transport executes a command", input: "ext::sh -c whoami", wantErr: true},
		{name: "ext transport uppercase", input: "EXT::sh -c whoami", wantErr: true},
		{name: "fd transport", input: "fd::7", wantErr: true},
		{name: "newline", input: "https://example.com\nrm -rf", wantErr: true},
		{name: "bare word is not a remote", input: "origin", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateRemote(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateRemote(%q) = nil, want an error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateRemote(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

func TestValidateOutputDirectoryRejectsWindowsHazards(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "device name alone", input: "NUL"},
		{name: "device name lowercase", input: "nul"},
		{name: "device name in a segment", input: "projects/CON/widget"},
		{name: "device name with extension", input: "projects/nul.txt"},
		{name: "UNC share", input: "//server/share/widget"},
		{name: "UNC share backslashes", input: `\\server\share\widget`},
		{name: "device namespace", input: `\\.\NUL`},
		{name: "long path namespace", input: `\\?\C:\widget`},
		{name: "drive relative", input: "C:widget"},
		{name: "segment ending in a dot", input: "projects/widget."},
		{name: "segment ending in a space", input: "projects/widget "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := config.ValidateOutputDirectory(tt.input); err == nil {
				t.Fatalf("ValidateOutputDirectory(%q) = nil, want an error", tt.input)
			}
		})
	}
}
