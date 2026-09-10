package generator_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/filesystem"
	"github.com/manjunathrathod/claude-repo-factory/internal/generator"
	"github.com/manjunathrathod/claude-repo-factory/internal/lang"
)

// recordingWorkspace stands in for the filesystem so a test can prove the
// generator did not reach it at all.
type recordingWorkspace struct {
	calls []call
	err   error
}

type call struct{ base, name string }

func (r *recordingWorkspace) Create(_ context.Context, base, name string) (filesystem.Result, error) {
	r.calls = append(r.calls, call{base: base, name: name})
	if r.err != nil {
		return filesystem.Result{}, r.err
	}
	return filesystem.Result{Path: filepath.Join(base, name), Created: true}, nil
}

// validConfig returns a configuration the registry accepts, rooted at dir.
func validConfig(dir string) config.ProjectConfig {
	cfg := config.Default()
	cfg.ProjectName = "payment-api"
	cfg.Description = "Payment service"
	cfg.Language = "go"
	cfg.ProjectType = config.ProjectTypeAPI
	cfg.PackageManager = "gomod"
	cfg.OutputDirectory = dir
	cfg.SetOption(lang.OptGoModule, "github.com/acme/payment-api")
	return cfg
}

func TestPrepareCreatesTheProjectDirectoryInsideTheOutputDirectory(t *testing.T) {
	// The headline behaviour: OutputDirectory is the parent, ProjectName is
	// the directory made inside it.
	base := t.TempDir()
	g := generator.New(filesystem.Workspace{})

	got, err := g.Prepare(context.Background(), validConfig(base), lang.Registry())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	want := filepath.Join(base, "payment-api")
	if got.Path != want {
		t.Errorf("Path = %q, want %q", got.Path, want)
	}
	if !got.Created {
		t.Error("Created = false, want true")
	}
	if info, statErr := os.Stat(want); statErr != nil {
		t.Fatalf("stat: %v", statErr)
	} else if !info.IsDir() {
		t.Error("target is not a directory")
	}
}

func TestPrepareCreatesNothingForAnInvalidConfiguration(t *testing.T) {
	// The guarantee this package exists to hold: validation runs first, so a
	// bad configuration never reaches the filesystem.
	tests := []struct {
		name   string
		mutate func(c *config.ProjectConfig)
	}{
		{name: "empty project name", mutate: func(c *config.ProjectConfig) { c.ProjectName = "" }},
		{name: "project name with a separator", mutate: func(c *config.ProjectConfig) { c.ProjectName = "acme/widget" }},
		{name: "traversing project name", mutate: func(c *config.ProjectConfig) { c.ProjectName = "../escaped" }},
		{name: "reserved device name", mutate: func(c *config.ProjectConfig) { c.ProjectName = "nul" }},
		{name: "traversing output directory", mutate: func(c *config.ProjectConfig) { c.OutputDirectory = "../../etc" }},
		{name: "unknown language", mutate: func(c *config.ProjectConfig) { c.Language = "cobol" }},
		{name: "unknown project type", mutate: func(c *config.ProjectConfig) { c.ProjectType = "mainframe" }},
		{name: "package manager from another ecosystem", mutate: func(c *config.ProjectConfig) { c.PackageManager = "npm" }},
		{name: "control character in the description", mutate: func(c *config.ProjectConfig) { c.Description = "a\nb" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			cfg := validConfig(base)
			tt.mutate(&cfg)

			ws := &recordingWorkspace{}
			g := generator.New(ws)

			if _, err := g.Prepare(context.Background(), cfg, lang.Registry()); err == nil {
				t.Fatal("Prepare() = nil error, want the configuration to be rejected")
			}
			if len(ws.calls) != 0 {
				t.Errorf("the filesystem was asked to create %v despite invalid input", ws.calls)
			}
			entries, err := os.ReadDir(base)
			if err != nil {
				t.Fatalf("read base: %v", err)
			}
			if len(entries) != 0 {
				t.Errorf("Prepare created %d entries for an invalid configuration, want none", len(entries))
			}
		})
	}
}

func TestPrepareRequiresACatalog(t *testing.T) {
	// Without a catalog every language would validate, so a nil one must be
	// an error rather than a skipped check.
	ws := &recordingWorkspace{}
	g := generator.New(ws)

	if _, err := g.Prepare(context.Background(), validConfig(t.TempDir()), nil); err == nil {
		t.Fatal("Prepare() = nil error, want a nil catalog to be rejected")
	}
	if len(ws.calls) != 0 {
		t.Errorf("the filesystem was reached without a catalog: %v", ws.calls)
	}
}

func TestPrepareSplitsTheConfigurationIntoBaseAndName(t *testing.T) {
	base := t.TempDir()
	ws := &recordingWorkspace{}
	g := generator.New(ws)

	if _, err := g.Prepare(context.Background(), validConfig(base), lang.Registry()); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if len(ws.calls) != 1 {
		t.Fatalf("Create called %d times, want once", len(ws.calls))
	}
	got := ws.calls[0]
	if got.base != base {
		t.Errorf("base = %q, want %q", got.base, base)
	}
	if got.name != "payment-api" {
		t.Errorf("name = %q, want the project name", got.name)
	}
}

func TestPrepareDefaultsToTheWorkingDirectory(t *testing.T) {
	// An empty OutputDirectory means "here", which must resolve to an
	// absolute path rather than being passed through as "".
	cfg := validConfig("")
	ws := &recordingWorkspace{}
	g := generator.New(ws)

	if _, err := g.Prepare(context.Background(), cfg, lang.Registry()); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if len(ws.calls) != 1 {
		t.Fatalf("Create called %d times, want once", len(ws.calls))
	}
	if !filepath.IsAbs(ws.calls[0].base) {
		t.Errorf("base = %q, want an absolute path", ws.calls[0].base)
	}
}

func TestPrepareAdoptsAnExistingEmptyDirectory(t *testing.T) {
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "payment-api"), 0o755); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	g := generator.New(filesystem.Workspace{})
	got, err := g.Prepare(context.Background(), validConfig(base), lang.Registry())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if got.Created {
		t.Error("Created = true, want false when an empty directory was adopted")
	}
}

func TestPrepareRefusesANonEmptyDirectory(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "payment-api")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	g := generator.New(filesystem.Workspace{})
	_, err := g.Prepare(context.Background(), validConfig(base), lang.Registry())
	if !errors.Is(err, filesystem.ErrNotEmpty) {
		t.Fatalf("Prepare() error = %v, want ErrNotEmpty", err)
	}
}

func TestPrepareWrapsAFilesystemFailure(t *testing.T) {
	sentinel := errors.New("disk on fire")
	g := generator.New(&recordingWorkspace{err: sentinel})

	_, err := g.Prepare(context.Background(), validConfig(t.TempDir()), lang.Registry())
	if !errors.Is(err, sentinel) {
		t.Fatalf("Prepare() error = %v, want it to wrap the filesystem error", err)
	}
	if !strings.Contains(err.Error(), "create repository directory") {
		t.Errorf("error = %q, want it to say what failed", err)
	}
}

func TestPrepareHonoursACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ws := &recordingWorkspace{}
	g := generator.New(ws)

	if _, err := g.Prepare(ctx, validConfig(t.TempDir()), lang.Registry()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Prepare() error = %v, want context.Canceled", err)
	}
	if len(ws.calls) != 0 {
		t.Errorf("a cancelled call still reached the filesystem: %v", ws.calls)
	}
}

func TestPrepareWithoutAWorkspaceIsAnError(t *testing.T) {
	var g *generator.Generator
	if _, err := g.Prepare(context.Background(), validConfig(t.TempDir()), lang.Registry()); err == nil {
		t.Fatal("Prepare() = nil error, want a missing workspace to be reported")
	}
}

func TestPrepareWritesNoFiles(t *testing.T) {
	// This milestone creates a directory and nothing else.
	base := t.TempDir()
	g := generator.New(filesystem.Workspace{})

	got, err := g.Prepare(context.Background(), validConfig(base), lang.Registry())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	entries, err := os.ReadDir(got.Path)
	if err != nil {
		t.Fatalf("read created directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the created directory holds %d entries, want none", len(entries))
	}
}
