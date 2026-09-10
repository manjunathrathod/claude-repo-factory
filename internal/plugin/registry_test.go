package plugin_test

import (
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// fake is a minimal Language used to exercise the registry without depending
// on any real plugin.
type fake struct {
	desc plugin.Descriptor
}

func (f fake) Descriptor() plugin.Descriptor              { return f.desc }
func (f fake) Instructions(spec.Spec) plugin.Instructions { return plugin.Instructions{} }
func (f fake) Files(spec.Spec) ([]plugin.FileSpec, error) { return nil, plugin.ErrNotImplemented }
func (f fake) Validate(spec.Spec) error                   { return nil }

func stable(id string, aliases ...string) fake {
	return fake{desc: plugin.Descriptor{ID: id, DisplayName: id, Status: plugin.StatusStable, Aliases: aliases}}
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
					ID:                 "go",
					Status:             plugin.StatusStable,
					ProjectTypes:       []plugin.ProjectType{{ID: "cli"}},
					DefaultProjectType: "service",
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
	want := []string{"go", "python", "rust"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
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
	d := plugin.Descriptor{ProjectTypes: []plugin.ProjectType{{ID: "cli"}, {ID: "service"}}}

	if got := strings.Join(d.ProjectTypeIDs(), ","); got != "cli,service" {
		t.Errorf("ProjectTypeIDs() = %q, want cli,service", got)
	}
	if !d.HasProjectType("service") {
		t.Error("HasProjectType(service) = false, want true")
	}
	if d.HasProjectType("library") {
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
