package plugin_test

import (
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
)

// fake is a minimal Language used to exercise the registry without depending
// on any real plugin.
type fake struct {
	desc plugin.Descriptor
}

func (f fake) Descriptor() plugin.Descriptor                         { return f.desc }
func (f fake) Instructions(config.ProjectConfig) plugin.Instructions { return plugin.Instructions{} }
func (f fake) Files(config.ProjectConfig) ([]plugin.FileSpec, error) {
	return nil, plugin.ErrNotImplemented
}
func (f fake) Validate(config.ProjectConfig) error { return nil }

// stable builds a minimally complete stable plugin: the registry now requires
// project types and package managers from anything claiming to be stable.
func stable(id config.Language, aliases ...string) fake {
	return fake{desc: plugin.Descriptor{
		ID:          id,
		DisplayName: string(id),
		Status:      plugin.StatusStable,
		Aliases:     aliases,
		ProjectTypes: []plugin.ProjectType{
			{ID: config.ProjectTypeLibrary, DisplayName: "Library"},
			{ID: config.ProjectTypeCLI, DisplayName: "CLI"},
		},
		DefaultProjectType:    config.ProjectTypeLibrary,
		PackageManagers:       []config.PackageManager{"pm"},
		DefaultPackageManager: "pm",
	}}
}

func TestRegisterAndLookup(t *testing.T) {
	r := plugin.NewRegistry()
	if err := r.Register(stable("go", "golang")); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	for _, name := range []string{"go", "golang", "GO", "  Golang  "} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("Lookup(%q) = not found, want the go plugin", name)
		}
	}
	if _, ok := r.Lookup("rust"); ok {
		t.Error("Lookup(rust) found a plugin, want not found")
	}
	if _, ok := r.Lookup(""); ok {
		t.Error("Lookup on an empty name found a plugin, want not found")
	}
}

func TestRegisterRejectsBadPlugins(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*plugin.Registry) error
		wantErr string
	}{
		{
			name:    "nil language",
			setup:   func(r *plugin.Registry) error { return r.Register(nil) },
			wantErr: "nil language",
		},
		{
			name:    "empty id",
			setup:   func(r *plugin.Registry) error { return r.Register(stable("")) },
			wantErr: "empty id",
		},
		{
			name: "duplicate id",
			setup: func(r *plugin.Registry) error {
				if err := r.Register(stable("go")); err != nil {
					return err
				}
				return r.Register(stable("go"))
			},
			wantErr: "already registered",
		},
		{
			name: "alias claimed by another plugin",
			setup: func(r *plugin.Registry) error {
				if err := r.Register(stable("go", "gopher")); err != nil {
					return err
				}
				return r.Register(stable("rust", "gopher"))
			},
			wantErr: "already used by",
		},
		{
			name: "alias colliding with a language id",
			setup: func(r *plugin.Registry) error {
				if err := r.Register(stable("go")); err != nil {
					return err
				}
				return r.Register(stable("rust", "go"))
			},
			wantErr: "collides with a language id",
		},
		{
			name: "unknown default project type",
			setup: func(r *plugin.Registry) error {
				return r.Register(fake{desc: plugin.Descriptor{
					ID:                    "go",
					Status:                plugin.StatusStable,
					ProjectTypes:          []plugin.ProjectType{{ID: config.ProjectTypeCLI}},
					DefaultProjectType:    config.ProjectTypeAPI,
					PackageManagers:       []config.PackageManager{"pm"},
					DefaultPackageManager: "pm",
				}})
			},
			wantErr: "unknown default project type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup(plugin.NewRegistry())
			if err == nil {
				t.Fatalf("setup() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("setup() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestListIsSortedAndAvailableFiltersPlanned(t *testing.T) {
	r := plugin.NewRegistry()
	r.MustRegister(stable("python"))
	r.MustRegister(stable("go"))
	r.MustRegister(fake{desc: plugin.Descriptor{ID: "rust", Status: plugin.StatusPlanned}})

	ids := r.IDs()
	want := []config.Language{"go", "python", "rust"}
	if config.JoinLanguages(ids, ",") != config.JoinLanguages(want, ",") {
		t.Errorf("IDs() = %v, want %v", ids, want)
	}

	available := r.Available()
	if len(available) != 2 {
		t.Fatalf("Available() returned %d plugins, want 2", len(available))
	}
	for _, l := range available {
		if l.Descriptor().Status != plugin.StatusStable {
			t.Errorf("Available() returned %q with status %q", l.Descriptor().ID, l.Descriptor().Status)
		}
	}
}

func TestGetErrorListsTheAlternatives(t *testing.T) {
	r := plugin.NewRegistry()
	r.MustRegister(stable("go"))
	r.MustRegister(stable("python"))

	_, err := r.Get("cobol")
	if err == nil {
		t.Fatal("Get(cobol) = nil error, want a failure")
	}
	for _, want := range []string{"cobol", "go", "python"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Get(cobol) error = %v, want it to mention %q", err, want)
		}
	}
}

func TestMustRegisterPanicsOnDuplicate(t *testing.T) {
	r := plugin.NewRegistry()
	r.MustRegister(stable("go"))

	defer func() {
		if recover() == nil {
			t.Error("MustRegister on a duplicate did not panic")
		}
	}()
	r.MustRegister(stable("go"))
}

func TestDescriptorProjectTypeHelpers(t *testing.T) {
	d := plugin.Descriptor{ProjectTypes: []plugin.ProjectType{
		{ID: config.ProjectTypeCLI},
		{ID: config.ProjectTypeAPI},
	}}

	if got := config.JoinProjectTypes(d.ProjectTypeIDs(), ","); got != "cli,api" {
		t.Errorf("ProjectTypeIDs() = %q, want cli,api", got)
	}
	if !d.HasProjectType(config.ProjectTypeAPI) {
		t.Error("HasProjectType(api) = false, want true")
	}
	if d.HasProjectType(config.ProjectTypeLibrary) {
		t.Error("HasProjectType(library) = true, want false")
	}
}

func TestRegistryIsSafeForConcurrentUse(t *testing.T) {
	r := plugin.NewRegistry()
	r.MustRegister(stable("go"))

	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				r.Lookup("go")
				r.List()
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

func TestRegistryIsACatalog(t *testing.T) {
	r := plugin.NewRegistry()
	r.MustRegister(stable("go", "golang"))
	r.MustRegister(stable("python", "py"))
	r.MustRegister(fake{desc: plugin.Descriptor{ID: "rust", Status: plugin.StatusPlanned}})

	t.Run("Languages excludes planned plugins", func(t *testing.T) {
		// A planned language is announced in listings but must never pass
		// validation, so the catalog view must not include it.
		got := config.JoinLanguages(r.Languages(), ",")
		if got != "go,python" {
			t.Fatalf("Languages() = %q, want go,python", got)
		}
	})

	t.Run("Resolve maps an alias to the canonical id", func(t *testing.T) {
		tests := map[string]config.Language{
			"golang": "go",
			"GOLANG": "go",
			"  py  ": "python",
			"go":     "go",
		}
		for input, want := range tests {
			got, ok := r.Resolve(input)
			if !ok {
				t.Errorf("Resolve(%q) = not found", input)
				continue
			}
			if got != want {
				t.Errorf("Resolve(%q) = %q, want %q", input, got, want)
			}
		}
	})

	t.Run("Resolve rejects the unknown", func(t *testing.T) {
		if _, ok := r.Resolve("cobol"); ok {
			t.Error("Resolve(cobol) = found, want not found")
		}
		if _, ok := r.Resolve(""); ok {
			t.Error("Resolve on an empty name = found, want not found")
		}
	})

	t.Run("ProjectTypes and PackageManagers of a known language", func(t *testing.T) {
		if got := config.JoinProjectTypes(r.ProjectTypes("go"), ","); got != "library,cli" {
			t.Errorf("ProjectTypes(go) = %q, want library,cli", got)
		}
		if got := config.JoinPackageManagers(r.PackageManagers("go"), ","); got != "pm" {
			t.Errorf("PackageManagers(go) = %q, want pm", got)
		}
	})

	t.Run("an unknown language yields nothing rather than panicking", func(t *testing.T) {
		if got := r.ProjectTypes("cobol"); got != nil {
			t.Errorf("ProjectTypes(cobol) = %v, want nil", got)
		}
		if got := r.PackageManagers("cobol"); got != nil {
			t.Errorf("PackageManagers(cobol) = %v, want nil", got)
		}
	})

	t.Run("PackageManagers cannot be mutated through the catalog", func(t *testing.T) {
		got := r.PackageManagers("go")
		if len(got) == 0 {
			t.Fatal("PackageManagers(go) returned nothing")
		}
		got[0] = "tampered"

		if again := r.PackageManagers("go"); again[0] != "pm" {
			t.Fatal("the catalog handed out a slice aliasing the descriptor")
		}
	})
}

func TestRegisterRejectsIncompleteStablePlugins(t *testing.T) {
	// A stable plugin that cannot answer the catalog would fail validation at
	// generation time; catching it at start-up is the whole point.
	base := func() plugin.Descriptor {
		return plugin.Descriptor{
			ID:                    "go",
			Status:                plugin.StatusStable,
			ProjectTypes:          []plugin.ProjectType{{ID: config.ProjectTypeCLI}},
			DefaultProjectType:    config.ProjectTypeCLI,
			PackageManagers:       []config.PackageManager{"pm"},
			DefaultPackageManager: "pm",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*plugin.Descriptor)
		wantErr string
	}{
		{name: "complete", mutate: nil},
		{
			name:    "no project types",
			mutate:  func(d *plugin.Descriptor) { d.ProjectTypes = nil; d.DefaultProjectType = "" },
			wantErr: "declares no project types",
		},
		{
			name:    "no package managers",
			mutate:  func(d *plugin.Descriptor) { d.PackageManagers = nil; d.DefaultPackageManager = "" },
			wantErr: "declares no package managers",
		},
		{
			name:    "default package manager not in the set",
			mutate:  func(d *plugin.Descriptor) { d.DefaultPackageManager = "other" },
			wantErr: "unknown default package manager",
		},
		{
			name: "project type outside the factory vocabulary",
			mutate: func(d *plugin.Descriptor) {
				d.ProjectTypes = []plugin.ProjectType{{ID: "service"}}
				d.DefaultProjectType = "service"
			},
			wantErr: "not in the factory vocabulary",
		},
		{
			name:    "no default project type",
			mutate:  func(d *plugin.Descriptor) { d.DefaultProjectType = "" },
			wantErr: "declares no default project type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := base()
			if tt.mutate != nil {
				tt.mutate(&d)
			}

			err := plugin.NewRegistry().Register(fake{desc: d})

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Register() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Register() = nil, want an error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Register() = %v, want an error containing %q", err, tt.wantErr)
			}
		})
	}
}

// A planned plugin is held to a lighter standard: it exists to reserve a name.
func TestRegisterAcceptsAMinimalPlannedPlugin(t *testing.T) {
	err := plugin.NewRegistry().Register(fake{desc: plugin.Descriptor{
		ID:     "rust",
		Status: plugin.StatusPlanned,
	}})
	if err != nil {
		t.Fatalf("Register() = %v, want nil", err)
	}
}
