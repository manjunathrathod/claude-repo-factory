package gitutil_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/gitutil"
)

// recorder is a Runner that records invocations and replays canned results,
// so command construction can be asserted without a real repository.
type recorder struct {
	calls  [][]string
	dirs   []string
	output string
	err    error
}

func (r *recorder) Run(_ context.Context, dir string, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	r.dirs = append(r.dirs, dir)
	return r.output, r.err
}

func (r *recorder) last() string {
	if len(r.calls) == 0 {
		return ""
	}
	return strings.Join(r.calls[len(r.calls)-1], " ")
}

func TestCommandConstruction(t *testing.T) {
	tests := []struct {
		name string
		call func(*gitutil.Client) error
		want string
	}{
		{
			name: "init with an explicit branch",
			call: func(c *gitutil.Client) error { return c.Init(context.Background(), "/repo", "main") },
			want: "init --initial-branch=main",
		},
		{
			name: "init without a branch defers to git",
			call: func(c *gitutil.Client) error { return c.Init(context.Background(), "/repo", "  ") },
			want: "init",
		},
		{
			name: "add all",
			call: func(c *gitutil.Client) error { return c.AddAll(context.Background(), "/repo") },
			want: "add --all",
		},
		{
			name: "commit",
			call: func(c *gitutil.Client) error { return c.Commit(context.Background(), "/repo", "init: scaffold") },
			want: "commit -m init: scaffold",
		},
		{
			name: "add remote",
			call: func(c *gitutil.Client) error {
				return c.AddRemote(context.Background(), "/repo", "git@github.com:acme/widget.git")
			},
			want: "remote add origin git@github.com:acme/widget.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{}
			client := &gitutil.Client{Runner: rec}

			if err := tt.call(client); err != nil {
				t.Fatalf("call error = %v", err)
			}
			if got := rec.last(); got != tt.want {
				t.Errorf("git %s, want git %s", got, tt.want)
			}
			if rec.dirs[0] != "/repo" {
				t.Errorf("ran in %q, want /repo", rec.dirs[0])
			}
		})
	}
}

func TestCommitMessageIsPassedAsASingleArgument(t *testing.T) {
	rec := &recorder{}
	client := &gitutil.Client{Runner: rec}

	// A message containing shell metacharacters must survive untouched;
	// this is why the wrapper never builds a shell string.
	message := "fix: handle $PATH & \"quotes\"; rm -rf /"
	if err := client.Commit(context.Background(), "/repo", message); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	args := rec.calls[0]
	if len(args) != 3 || args[2] != message {
		t.Fatalf("Commit() args = %q, want the message as one argument", args)
	}
}

func TestIsRepo(t *testing.T) {
	tests := []struct {
		name   string
		output string
		err    error
		want   bool
	}{
		{name: "inside a work tree", output: "true\n", want: true},
		{name: "outside a work tree", output: "false", want: false},
		{name: "git failed", err: errors.New("not a repository"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &gitutil.Client{Runner: &recorder{output: tt.output, err: tt.err}}
			if got := client.IsRepo(context.Background(), "/repo"); got != tt.want {
				t.Errorf("IsRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAvailablePropagatesFailure(t *testing.T) {
	client := &gitutil.Client{Runner: &recorder{err: gitutil.ErrGitMissing}}

	err := client.Available(context.Background())
	if !errors.Is(err, gitutil.ErrGitMissing) {
		t.Fatalf("Available() = %v, want ErrGitMissing", err)
	}
}

func TestExecRunnerReportsAMissingExecutable(t *testing.T) {
	runner := gitutil.ExecRunner{Bin: "definitely-not-a-real-git-binary"}

	_, err := runner.Run(context.Background(), "", "--version")
	if err == nil {
		t.Fatal("Run() = nil error, want a failure for a missing executable")
	}
}
