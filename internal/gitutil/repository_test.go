package gitutil_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/gitutil"
)

// scripted is a Runner that answers each git invocation from a table keyed by
// the first argument, so a test can drive Repository without a git
// installation and still assert on exactly what was asked for.
type scripted struct {
	calls   [][]string
	dirs    []string
	replies map[string]reply
}

type reply struct {
	out string
	err error
}

func newScripted() *scripted {
	return &scripted{replies: map[string]reply{
		// A working git by default: --version succeeds, the directory is not
		// yet a repository, init succeeds.
		"--version": {out: "git version 2.45.0"},
		"rev-parse": {out: "false"},
		"init":      {out: "Initialized empty Git repository"},
	}}
}

func (s *scripted) Run(_ context.Context, dir string, args ...string) (string, error) {
	s.calls = append(s.calls, args)
	s.dirs = append(s.dirs, dir)
	if len(args) == 0 {
		return "", nil
	}
	r, ok := s.replies[args[0]]
	if !ok {
		return "", nil
	}
	return r.out, r.err
}

// argsFor returns the invocation whose first argument is name.
func (s *scripted) argsFor(name string) []string {
	for _, c := range s.calls {
		if len(c) > 0 && c[0] == name {
			return c
		}
	}
	return nil
}

// dirFor returns the working directory used for the named invocation.
func (s *scripted) dirFor(name string) string {
	for i, c := range s.calls {
		if len(c) > 0 && c[0] == name {
			return s.dirs[i]
		}
	}
	return ""
}

func TestInitRunsAFixedCommandInTheGivenDirectory(t *testing.T) {
	dir := t.TempDir()
	run := newScripted()

	if err := gitutil.NewRepositoryWith(run).Init(context.Background(), dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got := run.argsFor("init")
	if got == nil {
		t.Fatalf("git init was never invoked; calls = %v", run.calls)
	}
	// Fixed executable, fixed arguments, no shell metacharacters, and the
	// branch passed as its own argument rather than interpolated.
	want := []string{"init", "--initial-branch=main"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("git args = %v, want %v", got, want)
	}
	if d := run.dirFor("init"); d != dir {
		t.Errorf("git init ran in %q, want %q", d, dir)
	}
}

func TestInitOmitsTheBranchFlagWhenNoBranchIsGiven(t *testing.T) {
	for _, branch := range []string{"", "   "} {
		t.Run("branch="+branch, func(t *testing.T) {
			run := newScripted()
			if err := gitutil.NewRepositoryWith(run).Init(context.Background(), t.TempDir(), branch); err != nil {
				t.Fatalf("Init() error = %v", err)
			}
			got := strings.Join(run.argsFor("init"), " ")
			if got != "init" {
				t.Errorf("git args = %q, want %q", got, "init")
			}
		})
	}
}

func TestInitNeverBuildsAShellString(t *testing.T) {
	// Every argument must arrive as its own slice element. A branch name
	// carrying shell metacharacters is passed through untouched and inert,
	// because there is no shell to interpret it.
	const hostile = `main; rm -rf / #$(whoami)`
	run := newScripted()

	if err := gitutil.NewRepositoryWith(run).Init(context.Background(), t.TempDir(), hostile); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got := run.argsFor("init")
	if len(got) != 2 {
		t.Fatalf("git args = %v, want exactly two elements", got)
	}
	if got[1] != "--initial-branch="+hostile {
		t.Errorf("args[1] = %q, want the branch as a single unsplit argument", got[1])
	}
}

func TestInitRejectsAnUnusableWorkingDirectory(t *testing.T) {
	existing := t.TempDir()
	file := filepath.Join(existing, "a-file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	tests := []struct {
		name string
		dir  string
	}{
		{name: "empty", dir: ""},
		{name: "relative", dir: "some/dir"},
		{name: "relative dot", dir: "."},
		{name: "missing", dir: filepath.Join(existing, "not-there")},
		{name: "a file", dir: file},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := newScripted()

			err := gitutil.NewRepositoryWith(run).Init(context.Background(), tt.dir, "main")
			if !errors.Is(err, gitutil.ErrWorkingDirectory) {
				t.Fatalf("Init(%q) error = %v, want ErrWorkingDirectory", tt.dir, err)
			}
			// The directory is checked before git is invoked at all.
			if len(run.calls) != 0 {
				t.Errorf("git ran despite an unusable directory: %v", run.calls)
			}
		})
	}
}

func TestInitRejectsASymlinkedWorkingDirectory(t *testing.T) {
	// Running in a link would run git somewhere other than the directory that
	// was created and reported to the user.
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows needs elevation")
	}

	root := t.TempDir()
	actual := filepath.Join(root, "actual")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(actual, 0o755); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := os.Symlink(actual, link); err != nil {
		t.Fatalf("prepare symlink: %v", err)
	}

	run := newScripted()
	err := gitutil.NewRepositoryWith(run).Init(context.Background(), link, "main")
	if !errors.Is(err, gitutil.ErrWorkingDirectory) {
		t.Fatalf("Init() error = %v, want ErrWorkingDirectory", err)
	}
	if len(run.calls) != 0 {
		t.Errorf("git ran in a symlinked directory: %v", run.calls)
	}
}

func TestInitReportsMissingGitHelpfully(t *testing.T) {
	run := newScripted()
	run.replies["--version"] = reply{err: gitutil.ErrGitMissing}

	err := gitutil.NewRepositoryWith(run).Init(context.Background(), t.TempDir(), "main")
	if !errors.Is(err, gitutil.ErrGitMissing) {
		t.Fatalf("Init() error = %v, want ErrGitMissing", err)
	}

	msg := err.Error()
	// The message must tell the user how to proceed.
	for _, want := range []string{"git-scm.com", "--no-git"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message does not mention %q:\n%s", want, msg)
		}
	}
	// And it must not leak the environment.
	for _, leak := range []string{os.Getenv("PATH"), "PATH="} {
		if leak != "" && strings.Contains(msg, leak) {
			t.Errorf("message leaks environment detail:\n%s", msg)
		}
	}
	if len(run.argsFor("init")) != 0 {
		t.Error("git init was attempted despite git being unavailable")
	}
}

func TestInitRefusesAnExistingRepository(t *testing.T) {
	run := newScripted()
	run.replies["rev-parse"] = reply{out: "true"}

	err := gitutil.NewRepositoryWith(run).Init(context.Background(), t.TempDir(), "main")
	if !errors.Is(err, gitutil.ErrAlreadyRepository) {
		t.Fatalf("Init() error = %v, want ErrAlreadyRepository", err)
	}
	if len(run.argsFor("init")) != 0 {
		t.Error("git init ran inside an existing repository")
	}
}

func TestInitSurfacesAGitFailure(t *testing.T) {
	sentinel := errors.New("disk full")
	run := newScripted()
	run.replies["init"] = reply{err: sentinel}

	err := gitutil.NewRepositoryWith(run).Init(context.Background(), t.TempDir(), "main")
	if !errors.Is(err, sentinel) {
		t.Fatalf("Init() error = %v, want it to wrap the git failure", err)
	}
}

func TestInitHonoursACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	run := newScripted()
	err := gitutil.NewRepositoryWith(run).Init(ctx, t.TempDir(), "main")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Init() error = %v, want context.Canceled", err)
	}
	if len(run.calls) != 0 {
		t.Errorf("git ran after cancellation: %v", run.calls)
	}
}

func TestAvailableReportsAWorkingGit(t *testing.T) {
	if err := gitutil.NewRepositoryWith(newScripted()).Available(context.Background()); err != nil {
		t.Fatalf("Available() error = %v, want nil", err)
	}
}

func TestValidateWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{name: "existing absolute directory", dir: dir},
		{name: "empty", dir: "", wantErr: true},
		{name: "relative", dir: "rel", wantErr: true},
		{name: "missing", dir: filepath.Join(dir, "nope"), wantErr: true},
		{name: "file", dir: file, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gitutil.ValidateWorkingDirectory(tt.dir)
			if tt.wantErr && !errors.Is(err, gitutil.ErrWorkingDirectory) {
				t.Fatalf("ValidateWorkingDirectory(%q) = %v, want ErrWorkingDirectory", tt.dir, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateWorkingDirectory(%q) = %v, want nil", tt.dir, err)
			}
		})
	}
}

// --- integration, only when a real git is present ---

// requireGit skips the test when git is not installed, so the suite still
// passes on a machine without it.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func TestIntegrationInitCreatesDotGitInTheGivenDirectoryOnly(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	target := filepath.Join(root, "payment-api")
	sibling := filepath.Join(root, "sibling")
	for _, d := range []string{target, sibling} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatalf("prepare: %v", err)
		}
	}

	if err := gitutil.NewRepository().Init(context.Background(), target, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// .git exists in the target...
	info, err := os.Stat(filepath.Join(target, ".git"))
	if err != nil {
		t.Fatalf("stat .git: %v", err)
	}
	if !info.IsDir() {
		t.Error(".git is not a directory")
	}
	// ...and nowhere else.
	for _, d := range []string{root, sibling} {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			t.Errorf("git init created .git in %s", d)
		}
	}
}

func TestIntegrationInitUsesTheRequestedBranch(t *testing.T) {
	requireGit(t)

	dir := t.TempDir()
	if err := gitutil.NewRepository().Init(context.Background(), dir, "trunk"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	head, err := os.ReadFile(filepath.Join(dir, ".git", "HEAD"))
	if err != nil {
		t.Fatalf("read HEAD: %v", err)
	}
	if !strings.Contains(string(head), "refs/heads/trunk") {
		t.Errorf("HEAD = %q, want it to point at trunk", strings.TrimSpace(string(head)))
	}
}

func TestIntegrationInitCreatesNoCommits(t *testing.T) {
	// This milestone initialises and stops.
	requireGit(t)

	dir := t.TempDir()
	if err := gitutil.NewRepository().Init(context.Background(), dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cmd := exec.CommandContext(context.Background(), "git", "rev-list", "--count", "--all")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list: %v: %s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "0" {
		t.Errorf("commit count = %s, want 0", got)
	}
}

func TestIntegrationInitConfiguresNoRemotes(t *testing.T) {
	requireGit(t)

	dir := t.TempDir()
	if err := gitutil.NewRepository().Init(context.Background(), dir, "main"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cmd := exec.CommandContext(context.Background(), "git", "remote")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git remote: %v: %s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "" {
		t.Errorf("remotes = %q, want none", got)
	}
}

func TestIntegrationInitRefusesToReinitialise(t *testing.T) {
	requireGit(t)

	dir := t.TempDir()
	repo := gitutil.NewRepository()
	if err := repo.Init(context.Background(), dir, "main"); err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	if err := repo.Init(context.Background(), dir, "main"); !errors.Is(err, gitutil.ErrAlreadyRepository) {
		t.Fatalf("second Init() error = %v, want ErrAlreadyRepository", err)
	}
}
