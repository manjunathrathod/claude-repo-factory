// Package cli wires the Cobra command tree.
//
// Commands never talk to the terminal or the registry directly: everything
// they need arrives through an App, which makes each command testable by
// substituting writers and a scripted prompt asker.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/filesystem"
	"github.com/manjunathrathod/claude-repo-factory/internal/generator"
	"github.com/manjunathrathod/claude-repo-factory/internal/gitutil"
	"github.com/manjunathrathod/claude-repo-factory/internal/lang"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/prompt"
	"github.com/manjunathrathod/claude-repo-factory/internal/version"
)

// Preparer creates the repository workspace for a validated configuration.
//
// It is declared here, where the command layer consumes it, so a test can
// substitute a double and assert on what the command asked for without
// touching a real filesystem. *generator.Generator satisfies it.
type Preparer interface {
	Prepare(ctx context.Context, cfg config.ProjectConfig, catalog config.Catalog) (generator.Result, error)
}

// App carries the dependencies shared by every command.
type App struct {
	Registry *plugin.Registry
	Asker    prompt.Asker
	Out      io.Writer
	Err      io.Writer
	// Generator creates the repository directory and initialises git. There is
	// deliberately no fallback for a nil Generator: this is the one dependency
	// whose failure mode is writing to a real disk, so forgetting to wire it
	// must fail loudly in a test rather than quietly reach the filesystem.
	// NewApp always sets it, and cli_test always injects a double.
	Generator Preparer
}

// NewApp returns an App wired to the built-in language registry, the
// interactive prompt implementation, the real filesystem and the real git.
func NewApp() *App {
	return &App{
		Registry:  lang.Registry(),
		Asker:     prompt.Survey{},
		Out:       os.Stdout,
		Err:       os.Stderr,
		Generator: generator.New(filesystem.Workspace{}, gitutil.NewRepository()),
	}
}

const rootLong = `claude-repo-factory creates fresh Git repositories that are configured for
professional software development and for Claude Code from the first commit.

A generated repository carries the engineering scaffolding a team would
otherwise assemble by hand: CLAUDE.md, .claude agents and workflows, CI,
documentation and test structure, coding standards, and security, testing,
architecture and Git instructions.

Language support is provided by plugins, so a new language or framework can
be added without changing the generator.`

// NewRootCommand builds the command tree.
func NewRootCommand(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:           "claude-repo-factory",
		Short:         "Create Git repositories configured for professional development and Claude Code",
		Long:          rootLong,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Get().Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.SetOut(app.Out)
	root.SetErr(app.Err)
	root.SetVersionTemplate("{{.Version}}\n")
	root.CompletionOptions.HiddenDefaultCmd = true

	root.AddCommand(
		newNewCommand(app),
		newLanguagesCommand(app),
		newVersionCommand(),
	)
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute(args []string) int {
	app := NewApp()
	root := NewRootCommand(app)
	root.SetArgs(args)

	if err := root.Execute(); err != nil {
		if errors.Is(err, prompt.ErrInterrupted) {
			fmt.Fprintln(app.Err, "cancelled")
			return 130
		}
		fmt.Fprintf(app.Err, "error: %v\n", err)
		return 1
	}
	return 0
}
