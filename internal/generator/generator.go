// Package generator turns a validated ProjectConfig into a repository on disk.
//
// It owns one rule that the rest of the tool depends on: nothing is created
// until the configuration has passed validation. The CLI validates before it
// shows the user a summary, but the CLI is not the only possible caller, so
// the guarantee is enforced here rather than assumed.
//
// This milestone creates the repository directory and stops. File generation
// and Git initialisation are later milestones; when they land they belong
// behind this same boundary, so a caller keeps talking to one thing.
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

// Result describes what preparing a workspace produced.
type Result struct {
	// Path is the absolute path of the repository directory.
	Path string
	// Created reports whether the directory was made, as opposed to an
	// existing empty directory being adopted.
	Created bool
}

// Generator creates repositories.
//
// A zero Generator is not usable: it needs a Workspace. Use New.
type Generator struct {
	workspace Workspace
}

// New returns a Generator that creates repositories through the given
// workspace.
func New(w Workspace) *Generator {
	return &Generator{workspace: w}
}

// Prepare validates cfg and creates the repository directory.
//
// The directory is created only after validation succeeds, so an invalid
// configuration never leaves anything behind. catalog answers which languages
// and project types this build supports; it is required, because validating
// without one would silently accept anything.
//
// Prepare writes no files and does not initialise Git.
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
	// it again from its own rules — the two packages do not share a
	// definition of "safe", and neither should have to trust the other.
	created, err := g.workspace.Create(ctx, base, cfg.ProjectName)
	if err != nil {
		return Result{}, fmt.Errorf("create repository directory: %w", err)
	}
	return Result{Path: created.Path, Created: created.Created}, nil
}
