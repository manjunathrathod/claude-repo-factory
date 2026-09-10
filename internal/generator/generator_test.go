package generator_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/filesystem"
	"github.com/manjunathrathod/claude-repo-factory/internal/generator"
	"github.com/manjunathrathod/claude-repo-factory/internal/gitutil"
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

// recordingGit records the directory git was asked to initialise.
type recordingGit struct {
	dirs     []string
	branches []string
	err      error
}

func (r *recordingGit) Init(_ context.Context, dir, branch string) error {
	r.dirs = append(r.dirs, dir)
	r.branches = append(r.branches, branch)
	return r.err
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
	g := generator.New(filesystem.Workspace{}, &recordingGit{})

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
	// bad configuration never reaches the filesystem or git.
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
			git := &recordingGit{}

			if _, err := generator.New(ws, git).Prepare(context.Background(), cfg, lang.Registry()); err == nil {
				t.Fatal("Prepare() = nil error, want the configuration to be rejected")
			}
			if len(ws.calls) != 0 {
				t.Errorf("the filesystem was asked to create %v despite invalid input", ws.calls)
			}
			if len(git.dirs) != 0 {
				t.Errorf("git ran despite invalid input: %v", git.dirs)
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
	ws := &recordingWorkspace{}
	g := generator.New(ws, &recordingGit{})

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

	if _, err := generator.New(ws, &recordingGit{}).Prepare(context.Background(), validConfig(base), lang.Registry()); err != nil {
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
	ws := &recordingWorkspace{}

	if _, err := generator.New(ws, &recordingGit{}).Prepare(context.Background(), validConfig(""), lang.Registry()); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if len(ws.calls) != 1 {
		t.Fatalf("Create called %d times, want once", len(ws.calls))
	}
	if !filepath.IsAbs(ws.calls[0].base) {
		t.Errorf("base = %q, want an absolute path", ws.calls[0].base)
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

	git := &recordingGit{}
	_, err := generator.New(filesystem.Workspace{}, git).Prepare(context.Background(), validConfig(base), lang.Registry())
	if !errors.Is(err, filesystem.ErrNotEmpty) {
		t.Fatalf("Prepare() error = %v, want ErrNotEmpty", err)
	}
	if len(git.dirs) != 0 {
		t.Errorf("git ran after the directory was refused: %v", git.dirs)
	}
}

// --- git initialisation ---

func TestPrepareInitialisesGitInTheDirectoryItCreated(t *testing.T) {
	base := t.TempDir()
	git := &recordingGit{}

	got, err := generator.New(filesystem.Workspace{}, git).Prepare(context.Background(), validConfig(base), lang.Registry())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	if len(git.dirs) != 1 {
		t.Fatalf("git init ran %d times, want once", len(git.dirs))
	}
	// Not the output directory, and not the working directory: the directory
	// that was just created.
	if git.dirs[0] != got.Path {
		t.Errorf("git ran in %q, want the created directory %q", git.dirs[0], got.Path)
	}
	if git.branches[0] != config.DefaultBranch {
		t.Errorf("branch = %q, want %q", git.branches[0], config.DefaultBranch)
	}
	if !got.GitInitialized {
		t.Error("GitInitialized = false, want true")
	}
}

func TestPrepareSkipsGitWhenInitializeGitIsFalse(t *testing.T) {
	base := t.TempDir()
	cfg := validConfig(base)
	cfg.InitializeGit = false
	git := &recordingGit{}

	got, err := generator.New(filesystem.Workspace{}, git).Prepare(context.Background(), cfg, lang.Registry())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	if len(git.dirs) != 0 {
		t.Errorf("git ran despite InitializeGit=false: %v", git.dirs)
	}
	if got.GitInitialized {
		t.Error("GitInitialized = true, want false")
	}
	// The directory is still created; only git is skipped.
	if _, statErr := os.Stat(got.Path); statErr != nil {
		t.Errorf("the project directory was not created: %v", statErr)
	}
}

func TestPrepareUsesTheConfiguredBranch(t *testing.T) {
	cfg := validConfig(t.TempDir())
	cfg.DefaultBranch = "trunk"
	git := &recordingGit{}

	if _, err := generator.New(filesystem.Workspace{}, git).Prepare(context.Background(), cfg, lang.Registry()); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if len(git.branches) != 1 || git.branches[0] != "trunk" {
		t.Errorf("branches = %v, want [trunk]", git.branches)
	}
}

func TestPrepareSurfacesAGitFailure(t *testing.T) {
	sentinel := errors.New("git exploded")
	git := &recordingGit{err: sentinel}

	_, err := generator.New(filesystem.Workspace{}, git).Prepare(context.Background(), validConfig(t.TempDir()), lang.Registry())
	if !errors.Is(err, sentinel) {
		t.Fatalf("Prepare() error = %v, want it to wrap the git failure", err)
	}
}

func TestPrepareFailsLoudlyWhenGitIsRequestedButUnavailable(t *testing.T) {
	// Asking for a git repository and silently not getting one would be the
	// worst outcome, so a missing client is an error rather than a skip.
	_, err := generator.New(filesystem.Workspace{}, nil).
		Prepare(context.Background(), validConfig(t.TempDir()), lang.Registry())
	if err == nil {
		t.Fatal("Prepare() = nil error, want a missing git client to be reported")
	}
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("error = %v, want it to name git", err)
	}
}

func TestPrepareWithoutGitClientSucceedsWhenGitIsNotWanted(t *testing.T) {
	cfg := validConfig(t.TempDir())
	cfg.InitializeGit = false

	if _, err := generator.New(filesystem.Workspace{}, nil).Prepare(context.Background(), cfg, lang.Registry()); err != nil {
		t.Fatalf("Prepare() error = %v, want nil when git was not requested", err)
	}
}

func TestPrepareHonoursACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ws := &recordingWorkspace{}
	git := &recordingGit{}

	if _, err := generator.New(ws, git).Prepare(ctx, validConfig(t.TempDir()), lang.Registry()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Prepare() error = %v, want context.Canceled", err)
	}
	if len(ws.calls) != 0 || len(git.dirs) != 0 {
		t.Error("a cancelled call still reached the filesystem or git")
	}
}

func TestPrepareWithoutAWorkspaceIsAnError(t *testing.T) {
	var g *generator.Generator
	if _, err := g.Prepare(context.Background(), validConfig(t.TempDir()), lang.Registry()); err == nil {
		t.Fatal("Prepare() = nil error, want a missing workspace to be reported")
	}
}

// --- integration, only when a real git is present ---

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func TestIntegrationPrepareProducesADotGitInTheProjectDirectory(t *testing.T) {
	// The end-to-end shape the feature asks for:
	//   payment-api/
	//   └── .git/
	requireGit(t)

	base := t.TempDir()
	g := generator.New(filesystem.Workspace{}, gitutil.NewRepository())

	got, err := g.Prepare(context.Background(), validConfig(base), lang.Registry())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !got.GitInitialized {
		t.Fatal("GitInitialized = false, want true")
	}

	info, statErr := os.Stat(filepath.Join(got.Path, ".git"))
	if statErr != nil {
		t.Fatalf("stat .git: %v", statErr)
	}
	if !info.IsDir() {
		t.Error(".git is not a directory")
	}
	// .git belongs to the project directory, not its parent.
	if _, err := os.Stat(filepath.Join(base, ".git")); err == nil {
		t.Error("git init leaked a .git into the output directory")
	}
}

func TestIntegrationPrepareWithoutGitLeavesNoDotGit(t *testing.T) {
	requireGit(t)

	cfg := validConfig(t.TempDir())
	cfg.InitializeGit = false

	got, err := generator.New(filesystem.Workspace{}, gitutil.NewRepository()).
		Prepare(context.Background(), cfg, lang.Registry())
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	entries, readErr := os.ReadDir(got.Path)
	if readErr != nil {
		t.Fatalf("read project directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("project directory holds %d entries, want none", len(entries))
	}
}
