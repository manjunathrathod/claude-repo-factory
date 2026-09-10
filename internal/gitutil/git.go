// Package gitutil is a thin, testable wrapper around the git command line.
//
// Every call goes through the Runner interface so that tests can exercise
// the command construction without a real repository, and so that the
// generator can be dry-run without touching the disk.
package gitutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrGitMissing is returned when the git executable cannot be found.
var ErrGitMissing = errors.New("gitutil: git executable not found in PATH")

// Runner executes a git invocation in dir and returns its combined output.
type Runner interface {
	Run(ctx context.Context, dir string, args ...string) (string, error)
}

// ExecRunner runs git as a real subprocess.
type ExecRunner struct {
	// Bin is the executable to invoke. Empty means "git" from PATH.
	Bin string
}

// Run implements Runner.
func (r ExecRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	bin := r.Bin
	if bin == "" {
		bin = "git"
	}
	// #nosec G204 -- args is an explicit slice built by this package, never a
	// shell string, which is precisely the injection defence this wrapper exists
	// to provide. bin is a fixed executable name or a test override.
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return text, ErrGitMissing
		}
		return text, fmt.Errorf("gitutil: git %s: %w: %s", strings.Join(args, " "), err, text)
	}
	return text, nil
}

// Client performs the small set of git operations the factory needs.
type Client struct {
	Runner Runner
}

// New returns a Client backed by the real git executable.
func New() *Client { return &Client{Runner: ExecRunner{}} }

// Version returns the output of git --version.
func (c *Client) Version(ctx context.Context) (string, error) {
	return c.Runner.Run(ctx, "", "--version")
}

// Available reports whether a usable git executable is present.
func (c *Client) Available(ctx context.Context) error {
	if _, err := c.Version(ctx); err != nil {
		return err
	}
	return nil
}

// Init creates a repository in dir with the given initial branch. An empty
// branch falls back to the git default.
func (c *Client) Init(ctx context.Context, dir, branch string) error {
	args := []string{"init"}
	if strings.TrimSpace(branch) != "" {
		args = append(args, "--initial-branch="+branch)
	}
	_, err := c.Runner.Run(ctx, dir, args...)
	return err
}

// IsRepo reports whether dir is inside a git work tree.
func (c *Client) IsRepo(ctx context.Context, dir string) bool {
	out, err := c.Runner.Run(ctx, dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// AddAll stages every change in dir.
func (c *Client) AddAll(ctx context.Context, dir string) error {
	_, err := c.Runner.Run(ctx, dir, "add", "--all")
	return err
}

// Commit creates a commit with the given message.
func (c *Client) Commit(ctx context.Context, dir, message string) error {
	_, err := c.Runner.Run(ctx, dir, "commit", "-m", message)
	return err
}

// AddRemote registers a remote named origin pointing at url.
func (c *Client) AddRemote(ctx context.Context, dir, url string) error {
	_, err := c.Runner.Run(ctx, dir, "remote", "add", "origin", url)
	return err
}
