package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/cli"
	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/filesystem"
	"github.com/manjunathrathod/claude-repo-factory/internal/generator"
	"github.com/manjunathrathod/claude-repo-factory/internal/gitutil"
)

// realGenerator returns the production generator, for the tests that must see
// the command touch a real filesystem and a real git.
func realGenerator() cli.Preparer {
	return generator.New(filesystem.Workspace{}, gitutil.NewRepository())
}

// requireGit skips a test when git is not installed, so the suite still passes
// on a machine without it.
func requireGit(t *testing.T) {
	t.Helper()
	if err := gitutil.NewRepository().Available(t.Context()); err != nil {
		t.Skip("git is not installed")
	}
}

func TestCreateMakesTheProjectDirectoryInsideTheOutputDirectory(t *testing.T) {
	dir := t.TempDir()

	out, _, err := runWith(t, nil, realGenerator(),
		"create", "payment-api", "-l", "go", "--dir", dir,
		"--set", "go_module=github.com/acme/payment-api", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	target := filepath.Join(dir, "payment-api")
	if info, statErr := os.Stat(target); statErr != nil {
		t.Fatalf("stat %s: %v\n%s", target, statErr, out)
	} else if !info.IsDir() {
		t.Fatalf("%s is not a directory", target)
	}
	if !strings.Contains(out, "Created ") || !strings.Contains(out, target) {
		t.Errorf("output does not report the created directory\n%s", out)
	}
}

// Regression: --dir means the parent, so the prompt default must be "." and
// never the project name. Defaulting to the name made the parent <cwd>/<name>
// and nested the repository at <cwd>/<name>/<name>.
func TestCreateWithoutADirectoryCreatesOneLevelInTheWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	out, _, err := runWith(t, nil, realGenerator(),
		"create", "payment-api", "-l", "go",
		"--set", "go_module=github.com/acme/payment-api", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read working directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "payment-api" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("working directory holds %v, want exactly [payment-api]", names)
	}
	// The repository directory must not contain a directory of its own name.
	if _, statErr := os.Stat(filepath.Join(dir, "payment-api", "payment-api")); statErr == nil {
		t.Error("the repository was nested inside a directory of its own name")
	}
}

func TestCreateInitialisesGitInTheProjectDirectory(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()

	out, _, err := runWith(t, nil, realGenerator(),
		"create", "payment-api", "-l", "go", "--dir", dir,
		"--set", "go_module=github.com/acme/payment-api", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	// payment-api/.git/ — the shape the feature asks for.
	gitDir := filepath.Join(dir, "payment-api", ".git")
	if info, statErr := os.Stat(gitDir); statErr != nil {
		t.Fatalf("stat %s: %v\n%s", gitDir, statErr, out)
	} else if !info.IsDir() {
		t.Error(".git is not a directory")
	}
	// And nowhere else.
	if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
		t.Error("git init leaked a .git into the output directory")
	}
	if !strings.Contains(out, "Initialised an empty Git repository") {
		t.Errorf("output does not report the git initialisation\n%s", out)
	}
}

func TestCreateWithNoGitLeavesNoRepository(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()

	out, _, err := runWith(t, nil, realGenerator(),
		"create", "payment-api", "-l", "go", "--dir", dir, "--no-git",
		"--set", "go_module=github.com/acme/payment-api", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	target := filepath.Join(dir, "payment-api")
	if _, statErr := os.Stat(target); statErr != nil {
		t.Fatalf("the project directory was not created: %v", statErr)
	}
	entries, readErr := os.ReadDir(target)
	if readErr != nil {
		t.Fatalf("read project directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("--no-git still produced %d entries, want none", len(entries))
	}
	if strings.Contains(out, "Initialised an empty Git repository") {
		t.Errorf("output claims git was initialised despite --no-git\n%s", out)
	}
}

func TestCreateRefusesANonEmptyProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "widget")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	_, _, err := runWith(t, nil, realGenerator(),
		"create", "widget", "-l", "go", "--dir", dir,
		"--set", "go_module=github.com/acme/widget", "--yes")
	if !errors.Is(err, filesystem.ErrNotEmpty) {
		t.Fatalf("create error = %v, want ErrNotEmpty", err)
	}
	if _, statErr := os.Stat(filepath.Join(target, "main.go")); statErr != nil {
		t.Errorf("create disturbed the existing directory: %v", statErr)
	}
	// The message must say how to recover, not just that it failed.
	for _, want := range []string{"different name", "different location", "empty the directory"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q:\n%v", want, err)
		}
	}
}

func TestCreateCreatesNothingWhenDeclined(t *testing.T) {
	dir := t.TempDir()
	asker := fullyScripted()
	asker.Answers[askOutputDir] = dir
	asker.Confirms[askConfirm] = false

	out, _, err := runWith(t, asker, realGenerator(), "create")
	if err != nil {
		t.Fatalf("declining is not an error, got %v", err)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("read output directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("declining still created %d entries\n%s", len(entries), out)
	}
}

func TestCreateCreatesNothingForAnInvalidConfiguration(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := runWith(t, nil, realGenerator(), "create", "nul", "-l", "go", "--dir", dir, "--yes"); err == nil {
		t.Fatal("create = nil error, want the reserved name to be rejected")
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("read output directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("an invalid configuration created %d entries, want none", len(entries))
	}
}

func TestCreateAnnouncesTheTargetBeforeAskingToConfirm(t *testing.T) {
	// The summary shows the parent, so the target has to be stated separately
	// or the user is confirming a path they were never shown.
	asker := fullyScripted()
	asker.Answers[askOutputDir] = "projects"

	out, _, err := runWith(t, asker, newFakeGenerator(), "create")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	at := strings.Index(out, "The repository will be created at ")
	if at < 0 {
		t.Fatalf("output never states the target\n%s", out)
	}
	if !strings.Contains(out[at:], filepath.Join("projects", "widget")) {
		t.Errorf("the stated target is not the project directory\n%s", out[at:])
	}
	if !slices.Contains(asker.Asked, askConfirm) {
		t.Fatal("the user was never asked to confirm")
	}
}

func TestCreateHandsTheGeneratorTheResolvedConfiguration(t *testing.T) {
	gen := newFakeGenerator()

	out, _, err := runWith(t, nil, gen,
		"create", "payment-api", "-l", "node", "-t", "api", "--package-manager", "pnpm",
		"--dir", "services", "--set", "npm_package=payment-api", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	if len(gen.configs) != 1 {
		t.Fatalf("generator called %d times, want once", len(gen.configs))
	}

	got := gen.configs[0]
	if got.ProjectName != "payment-api" {
		t.Errorf("ProjectName = %q", got.ProjectName)
	}
	if got.Language != "nodejs" {
		t.Errorf("Language = %q, want the canonical id", got.Language)
	}
	if got.ProjectType != config.ProjectTypeAPI {
		t.Errorf("ProjectType = %q", got.ProjectType)
	}
	if got.PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q", got.PackageManager)
	}
	if got.OutputDirectory != "services" {
		t.Errorf("OutputDirectory = %q", got.OutputDirectory)
	}
	if !got.InitializeGit {
		t.Error("InitializeGit = false, want true by default")
	}
}

func TestCreatePassesInitializeGitThrough(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "default", args: nil, want: true},
		{name: "no-git", args: []string{"--no-git"}, want: false},
		{name: "explicit no-git=false", args: []string{"--no-git=false"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := newFakeGenerator()
			args := append([]string{"create", "widget", "-l", "go", "--yes"}, tt.args...)

			if _, _, err := runWith(t, nil, gen, args...); err != nil {
				t.Fatalf("create error = %v", err)
			}
			if len(gen.configs) != 1 {
				t.Fatalf("generator called %d times, want once", len(gen.configs))
			}
			if got := gen.configs[0].InitializeGit; got != tt.want {
				t.Errorf("InitializeGit = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDoesNotReachTheGeneratorWhenDeclined(t *testing.T) {
	gen := newFakeGenerator()
	asker := fullyScripted()
	asker.Confirms[askConfirm] = false

	if _, _, err := runWith(t, asker, gen, "create"); err != nil {
		t.Fatalf("create error = %v", err)
	}
	if len(gen.configs) != 0 {
		t.Errorf("the generator ran despite the user declining: %v", gen.configs)
	}
}

func TestCreateSurfacesAGeneratorFailure(t *testing.T) {
	gen := newFakeGenerator()
	gen.err = errors.New("disk on fire")

	_, _, err := runWith(t, nil, gen, "create", "widget", "-l", "go", "--yes")
	if err == nil {
		t.Fatal("create = nil error, want the generator failure to surface")
	}
	if !strings.Contains(err.Error(), "disk on fire") {
		t.Errorf("error = %v, want it to carry the cause", err)
	}
}

func TestCreateReportsThePathTheGeneratorReturned(t *testing.T) {
	// The command must print what the generator actually created, not a path
	// it recomputed for itself.
	gen := newFakeGenerator()

	out, _, err := runWith(t, nil, gen, "create", "widget", "-l", "go", "--dir", "somewhere", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	if !strings.Contains(out, fakePath) {
		t.Errorf("output does not report the generator's path %q\n%s", fakePath, out)
	}
}

func TestCreateDistinguishesCreatedFromAdopted(t *testing.T) {
	tests := []struct {
		name    string
		created bool
		want    string
		unwant  string
	}{
		{name: "created", created: true, want: "Created ", unwant: "Using existing"},
		{name: "adopted", created: false, want: "Using existing empty directory", unwant: "Created "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := newFakeGenerator()
			gen.created = tt.created

			out, _, err := runWith(t, nil, gen, "create", "widget", "-l", "go", "--yes")
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("output is missing %q\n%s", tt.want, out)
			}
			if strings.Contains(out, tt.unwant) {
				t.Errorf("output wrongly contains %q\n%s", tt.unwant, out)
			}
		})
	}
}
