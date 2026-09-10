package lang_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/lang"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// wantStable is the set of languages this milestone promises to support.
var wantStable = []string{"go", "java", "nodejs", "python"}

// wantPlanned is the roadmap that must stay visible to users.
var wantPlanned = []string{"dotnet", "nextjs", "react", "rust", "terraform"}

func TestEveryPromisedLanguageIsRegistered(t *testing.T) {
	r := lang.Registry()

	for _, id := range wantStable {
		l, err := r.Get(id)
		if err != nil {
			t.Errorf("Get(%q) error = %v", id, err)
			continue
		}
		if got := l.Descriptor().Status; got != plugin.StatusStable {
			t.Errorf("%s status = %q, want %q", id, got, plugin.StatusStable)
		}
	}

	for _, id := range wantPlanned {
		l, err := r.Get(id)
		if err != nil {
			t.Errorf("Get(%q) error = %v", id, err)
			continue
		}
		if got := l.Descriptor().Status; got != plugin.StatusPlanned {
			t.Errorf("%s status = %q, want %q", id, got, plugin.StatusPlanned)
		}
	}
}

func TestAliasesResolve(t *testing.T) {
	tests := map[string]string{
		"golang":     "go",
		"node":       "nodejs",
		"typescript": "nodejs",
		"ts":         "nodejs",
		"py":         "python",
		"jvm":        "java",
		"tf":         "terraform",
		"c#":         "dotnet",
	}

	for alias, wantID := range tests {
		t.Run(alias, func(t *testing.T) {
			l, ok := lang.Registry().Lookup(alias)
			if !ok {
				t.Fatalf("Lookup(%q) = not found", alias)
			}
			if got := l.Descriptor().ID; got != wantID {
				t.Fatalf("Lookup(%q) = %q, want %q", alias, got, wantID)
			}
		})
	}
}

func TestStableLanguagesAreFullyDescribed(t *testing.T) {
	for _, id := range wantStable {
		t.Run(id, func(t *testing.T) {
			l, err := lang.Registry().Get(id)
			if err != nil {
				t.Fatalf("Get(%q) error = %v", id, err)
			}
			d := l.Descriptor()

			if d.DisplayName == "" || d.Summary == "" {
				t.Error("descriptor is missing a display name or summary")
			}
			if len(d.ProjectTypes) == 0 {
				t.Error("descriptor declares no project types")
			}
			if !d.HasProjectType(d.DefaultProjectType) {
				t.Errorf("default project type %q is not one of %v", d.DefaultProjectType, d.ProjectTypeIDs())
			}

			ins := l.Instructions(sampleSpec(id))
			for name, section := range map[string]string{
				"Toolchain":    ins.Toolchain,
				"Standards":    ins.Standards,
				"Testing":      ins.Testing,
				"Security":     ins.Security,
				"Architecture": ins.Architecture,
			} {
				if strings.TrimSpace(section) == "" {
					t.Errorf("instruction section %s is empty", name)
				}
			}
			if len(ins.Commands) == 0 {
				t.Error("no commands declared; CLAUDE.md and CI would have nothing to agree on")
			}
			for _, c := range ins.Commands {
				if c.Name == "" || c.Run == "" || c.Description == "" {
					t.Errorf("command %+v is incompletely declared", c)
				}
			}
			if !hasCommand(ins.Commands, "test") {
				t.Error("no test command declared")
			}
		})
	}
}

func TestFilesReportsNotImplemented(t *testing.T) {
	l, err := lang.Registry().Get("go")
	if err != nil {
		t.Fatalf("Get(go) error = %v", err)
	}

	files, err := l.Files(sampleSpec("go"))
	if files != nil {
		t.Errorf("Files() returned %d files, want none while generation is unimplemented", len(files))
	}
	if !errors.Is(err, plugin.ErrNotImplemented) {
		t.Fatalf("Files() error = %v, want plugin.ErrNotImplemented", err)
	}
	if !strings.Contains(err.Error(), "go") {
		t.Errorf("Files() error = %v, want it to name the language", err)
	}
}

func TestValidate(t *testing.T) {
	registry := lang.Registry()

	t.Run("accepts a complete spec", func(t *testing.T) {
		l, err := registry.Get("go")
		if err != nil {
			t.Fatalf("Get(go) error = %v", err)
		}
		if err := l.Validate(sampleSpec("go")); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	t.Run("rejects an unknown project type", func(t *testing.T) {
		l, err := registry.Get("go")
		if err != nil {
			t.Fatalf("Get(go) error = %v", err)
		}
		s := sampleSpec("go")
		s.ProjectType = "mainframe"
		err = l.Validate(s)
		if err == nil || !strings.Contains(err.Error(), "unknown project type") {
			t.Fatalf("Validate() = %v, want an unknown project type error", err)
		}
	})

	t.Run("rejects a missing required option", func(t *testing.T) {
		l, err := registry.Get("go")
		if err != nil {
			t.Fatalf("Get(go) error = %v", err)
		}
		s := sampleSpec("go")
		s.Options = map[string]string{}
		err = l.Validate(s)
		if err == nil || !strings.Contains(err.Error(), lang.OptGoModule) {
			t.Fatalf("Validate() = %v, want it to demand %q", err, lang.OptGoModule)
		}
	})

	t.Run("rejects a planned language", func(t *testing.T) {
		l, err := registry.Get("rust")
		if err != nil {
			t.Fatalf("Get(rust) error = %v", err)
		}
		err = l.Validate(sampleSpec("rust"))
		if err == nil || !strings.Contains(err.Error(), "not available yet") {
			t.Fatalf("Validate() = %v, want a not-available error", err)
		}
	})
}

func TestOptionsExposeDefaults(t *testing.T) {
	tests := map[string]struct {
		key  string
		name string
		want string
	}{
		"go":     {lang.OptGoModule, "widget", "github.com/example/widget"},
		"python": {lang.OptPythonPackage, "My-Widget", "my_widget"},
		"nodejs": {lang.OptPackageName, "My-Widget", "my-widget"},
		"java":   {lang.OptJavaGroupID, "widget", "com.example.widget"},
	}

	for id, tt := range tests {
		t.Run(id, func(t *testing.T) {
			l, err := lang.Registry().Get(id)
			if err != nil {
				t.Fatalf("Get(%q) error = %v", id, err)
			}
			opts := lang.Options(l)
			if len(opts) == 0 {
				t.Fatal("Options() returned nothing, want at least one option spec")
			}

			var found *lang.OptionSpec
			for i := range opts {
				if opts[i].Key == tt.key {
					found = &opts[i]
				}
				if opts[i].Prompt == "" || opts[i].Help == "" {
					t.Errorf("option %q has no prompt or help text", opts[i].Key)
				}
			}
			if found == nil {
				t.Fatalf("option %q is not declared", tt.key)
			}

			s := spec.Default()
			s.Name = tt.name
			if got := found.DefaultValue(s); got != tt.want {
				t.Errorf("DefaultValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOptionsOfANonDefinitionIsNil(t *testing.T) {
	if got := lang.Options(nil); got != nil {
		t.Errorf("Options(nil) = %v, want nil", got)
	}
}

func sampleSpec(language string) spec.Spec {
	s := spec.Default()
	s.Name = "widget"
	s.Language = language
	s.Options = map[string]string{
		lang.OptGoModule:       "github.com/acme/widget",
		lang.OptPythonPackage:  "widget",
		lang.OptPackageName:    "widget",
		lang.OptJavaGroupID:    "com.acme.widget",
		lang.OptJavaArtifactID: "widget",
	}
	return s
}

func hasCommand(cmds []plugin.Command, name string) bool {
	for _, c := range cmds {
		if c.Name == name {
			return true
		}
	}
	return false
}
