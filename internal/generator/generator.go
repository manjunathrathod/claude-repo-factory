// Package generator turns a validated ProjectConfig into a repository on disk.
//
// It owns one rule the rest of the tool depends on: nothing is created until
// the configuration has passed validation. The CLI validates before it shows
// the user a summary, but the CLI is not the only possible caller, so the
// guarantee is enforced here rather than assumed.
//
// This milestone creates the repository directory and, when the configuration
// asks for it, initialises git inside it. File generation is a later
// milestone; when it lands it belongs behind this same boundary, so a caller
// keeps talking to one thing.
package generator

import (
	"context"
	"fmt"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/filesystem"
)

// Workspace is the filesystem capability the generator needs.
//
// It is declared here, where it is consumed, so a test can substitute a double
// without the filesystem package knowing anything about generation.
// filesystem.Workspace satisfies it.
type Workspace interface {
	// Create makes the directory called name inside base and returns where it
	// is. An existing empty directory is adopted; a non-empty one is an error.
	Create(ctx context.Context, base, name string) (filesystem.Result, error)
}

// Git initialises a repository in a directory the generator has created.
//
// It is declared here for the same reason as Workspace, and is deliberately
// this small: the generator has no business committing, adding remotes or
// reaching a network, so nothing wider is reachable through it.
// *gitutil.Repository satisfies it.
type Git interface {
	// Init creates a git repository in dir with the given initial branch.
	Init(ctx context.Context, dir, branch string) error
}

// Result describes what preparing a workspace produced.
type Result struct {
	// Path is the absolute path of the repository directory.
	Path string
	// Created reports whether the directory was made, as opposed to an
	// existing empty directory being adopted.
	Created bool
	// GitInitialized reports whether a git repository was initialised. It is
	// false when the configuration asked for no git.
	GitInitialized bool
}

// Generator creates repositories.
//
// A zero Generator is not usable: it needs a Workspace. Use New.
type Generator struct {
	workspace Workspace
	git       Git
}

// New returns a Generator that creates repositories through the given
// workspace and initialises git through the given Git.
//
// A nil Git is permitted and means git initialisation is unavailable; a
// configuration that asks for it then fails rather than silently skipping,
// because "I asked for a git repository and did not get one" must never pass
// quietly.
func New(w Workspace, g Git) *Generator {
	return &Generator{workspace: w, git: g}
}

// Prepare validates cfg, creates the repository directory, and initialises git
// inside it when cfg.InitializeGit is set.
//
// The directory is created only after validation succeeds, so an invalid
// configuration never leaves anything behind. catalog answers which languages
// and project types this build supports; it is required, because validating
// without one would silently accept anything.
//
// Prepare writes no files, creates no commits and configures no remotes.
func (g *Generator) Prepare(ctx context.Context, cfg config.ProjectConfig, catalog config.Catalog) (Result, error) {
	if g == nil || g.workspace == nil {
		return Result{}, fmt.Errorf("generator: no workspace configured")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	// Validation first, and unconditionally. Everything below this line
	// touches the filesystem.
	if err := cfg.Validate(catalog); err != nil {
		return Result{}, fmt.Errorf("invalid project configuration: %w", err)
	}

	base, err := cfg.ResolvedOutputDirectory()
	if err != nil {
		return Result{}, fmt.Errorf("resolve output directory: %w", err)
	}

	// The project name is the directory name. It has already been through
	// config.ValidateProjectName above, and filesystem.ValidateSegment checks
	// it again from its own rules — the two packages do not share a definition
	// of "safe", and neither should have to trust the other.
	created, err := g.workspace.Create(ctx, base, cfg.ProjectName)
	if err != nil {
		return Result{}, fmt.Errorf("create repository directory: %w", err)
	}

	result := Result{Path: created.Path, Created: created.Created}
	if !cfg.InitializeGit {
		return result, nil
	}

	if g.git == nil {
		return result, fmt.Errorf("git initialisation was requested but no git client is configured")
	}
	// created.Path is absolute and was produced by the workspace, so git runs
	// in the directory that was just made and nowhere else.
	if err := g.git.Init(ctx, created.Path, cfg.DefaultBranch); err != nil {
		return result, fmt.Errorf("initialise git in %s: %w", created.Path, err)
	}
	result.GitInitialized = true
	return result, nil
}
