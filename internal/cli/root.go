// Package cli wires the Cobra command tree.
//
// Commands never talk to the terminal or the registry directly: everything
// they need arrives through an App, which makes each command testable by
// substituting writers and a scripted prompt asker.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/manjunathrathod/claude-repo-factory/internal/lang"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/prompt"
	"github.com/manjunathrathod/claude-repo-factory/internal/version"
)

// App carries the dependencies shared by every command.
type App struct {
	Registry *plugin.Registry
	Asker    prompt.Asker
	Out      io.Writer
	Err      io.Writer
}

// NewApp returns an App wired to the built-in language registry and the
// interactive prompt implementation.
func NewApp() *App {
	return &App{
		Registry: lang.Registry(),
		Asker:    prompt.Survey{},
		Out:      os.Stdout,
		Err:      os.Stderr,
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
