// Command claude-repo-factory creates fresh Git repositories that are
// configured for professional software development and for Claude Code.
//
// main stays thin on purpose: it only translates the CLI result into a
// process exit code. All behaviour lives in internal packages so it can be
// tested without spawning a process.
package main

import (
	"os"

	"github.com/manjunathrathod/claude-repo-factory/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
