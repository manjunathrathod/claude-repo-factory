package gitutil

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Failures the caller is expected to react to, rather than match on message
// text.
var (
	// ErrWorkingDirectory is returned when the directory git was asked to run
	// in is missing, is not a directory, or was given as a relative path.
	ErrWorkingDirectory = errors.New("gitutil: unusable working directory")
	// ErrAlreadyRepository is returned when the directory is already a git
	// work tree. Re-initialising is not destructive, but silently doing it
	// hides that the caller did not expect a repository to be there.
	ErrAlreadyRepository = errors.New("gitutil: directory is already a git repository")
)

// Repository initialises git repositories, and nothing else.
//
// It is a deliberately narrow front door onto Client: the factory's one job
// with git in this milestone is `git init` in a directory it has just created.
// Commits, remotes and anything reaching the network are later milestones, so
// they are not reachable from here.
//
// Every invocation is built as an explicit argument slice and handed to
// exec.CommandContext by ExecRunner. No shell is involved at any point, so
// there is no string for user input to escape from; the only user-controlled
// value that reaches git at all is the initial branch name, which
// config.Validate has already constrained.
type Repository struct {
	client *Client
}

// NewRepository returns a Repository backed by the real git executable.
func NewRepository() *Repository { return &Repository{client: New()} }

// NewRepositoryWith returns a Repository backed by the given runner, so a test
// can observe the exact arguments without a git installation.
func NewRepositoryWith(r Runner) *Repository { return &Repository{client: &Client{Runner: r}} }

// Available reports whether a usable git executable is present.
//
// The returned error is safe to show a user: it names the problem and what to
// do about it, and carries nothing from the environment — not PATH, not the
// resolved executable location, not the process environment.
func (r *Repository) Available(ctx context.Context) error {
	if r == nil || r.client == nil {
		return errors.New("gitutil: no client configured")
	}
	if err := r.client.Available(ctx); err != nil {
		if errors.Is(err, ErrGitMissing) {
			return fmt.Errorf("%w\n\nGit is required to initialise a repository. Install it from "+
				"https://git-scm.com/downloads, or re-run with --no-git to skip this step", ErrGitMissing)
		}
		return fmt.Errorf("git is installed but did not run: %w", err)
	}
	return nil
}

// Init creates a git repository in dir with the given initial branch. An empty
// branch falls back to git's own default.
//
// dir must be an existing absolute path to a directory. That is checked before
// git is invoked, so the command can never run somewhere the caller did not
// mean — a relative path would otherwise resolve against this process's
// working directory rather than the generated repository.
func (r *Repository) Init(ctx context.Context, dir, branch string) error {
	if r == nil || r.client == nil {
		return errors.New("gitutil: no client configured")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateWorkingDirectory(dir); err != nil {
		return err
	}
	if err := r.Available(ctx); err != nil {
		return err
	}
	// Refusing an existing repository keeps this operation honest: the factory
	// creates repositories, it does not adopt or re-initialise them.
	if r.client.IsRepo(ctx, dir) {
		return fmt.Errorf("%s: %w", dir, ErrAlreadyRepository)
	}
	if err := r.client.Init(ctx, dir, branch); err != nil {
		return fmt.Errorf("initialise git repository in %s: %w", dir, err)
	}
	return nil
}

// ValidateWorkingDirectory reports whether dir is safe to run git in.
//
// It requires an existing absolute path to a real directory. Absolute because
// a relative path resolves against this process's working directory, which is
// not where the repository is; existing because exec silently reports a
// confusing failure otherwise; a directory rather than a symlink to one so
// that the command runs where the caller believes it does.
func ValidateWorkingDirectory(dir string) error {
	if dir == "" {
		return fmt.Errorf("%w: no directory given", ErrWorkingDirectory)
	}
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("%w: %s is not an absolute path", ErrWorkingDirectory, dir)
	}

	info, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s does not exist", ErrWorkingDirectory, dir)
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrWorkingDirectory, dir, err)
	}
	// Lstat, not Stat: a symlink here would run git somewhere other than the
	// directory that was created and reported to the user.
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s is a symbolic link", ErrWorkingDirectory, dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory", ErrWorkingDirectory, dir)
	}
	return nil
}
