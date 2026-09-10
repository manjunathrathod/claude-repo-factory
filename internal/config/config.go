// Package config defines the project configuration model: the strongly typed,
// fully resolved description of a repository the factory has been asked to
// create.
//
// The model is deliberately free of both generation logic and CLI concerns. It
// is the single value that flows from the command layer (flags and interactive
// prompts) into language plugins. Validation lives here rather than in the CLI
// so that any caller — a command, a test, or a future API — gets the same
// answer about whether a configuration is safe to act on.
//
// The package knows no language names. Which languages, project types and
// package managers are valid is answered by a Catalog, which the plugin
// registry implements; see ADR 0003.
package config

import (
	"path/filepath"
	"sort"
	"strings"
)

// LicenseUnlicensed is the License value for a repository that intentionally
// ships without a licence file.
const LicenseUnlicensed = "none"

// DefaultBranch is the branch name used for freshly initialised repositories
// unless the caller overrides it.
const DefaultBranch = "main"

// ProjectConfig is the resolved description of a repository to generate.
//
// A zero ProjectConfig is not valid; start from Default and layer flags and
// prompt answers over it. Plugins receive it by value and must not mutate it.
type ProjectConfig struct {
	// ProjectName is the repository name. It becomes the directory name, so
	// ValidateProjectName constrains it tightly.
	ProjectName string
	// Description is the one-line summary used in README.md and CLAUDE.md.
	Description string
	// Author is the person or team recorded in the licence and documentation.
	Author string
	// License is an SPDX identifier, or LicenseUnlicensed for proprietary code.
	License string

	// Language selects the plugin that shapes the repository. It stores a
	// canonical plugin id, never an alias: resolve aliases with Catalog.Resolve
	// before assigning.
	Language Language
	// ProjectType is the shape of the repository, from the closed factory
	// vocabulary.
	ProjectType ProjectType
	// PackageManager is the dependency tool the repository will use. Valid
	// values depend on Language and come from the Catalog.
	PackageManager PackageManager

	// OutputDirectory is where the repository is created. Empty means "a
	// directory named ProjectName, under the working directory".
	OutputDirectory string
	// DefaultBranch is the initial branch name.
	DefaultBranch string
	// Remote is an optional Git remote URL registered as origin.
	Remote string

	// InitializeGit controls whether a Git repository is initialised.
	InitializeGit bool
	// IncludeClaude controls whether .claude configuration is generated.
	IncludeClaude bool
	// IncludeClaudeAgents controls whether specialist agents are generated.
	IncludeClaudeAgents bool
	// IncludeClaudeWorkflows controls whether reusable workflows are generated.
	IncludeClaudeWorkflows bool
	// IncludeGitHubActions controls whether CI workflows are generated.
	IncludeGitHubActions bool
	// IncludeDocs controls whether the docs structure is generated.
	IncludeDocs bool
	// IncludeTests controls whether the test structure is generated.
	IncludeTests bool
	// IncludePRTemplate controls whether a pull request template is generated.
	IncludePRTemplate bool
	// IncludeCodingStandards controls whether coding standards are generated.
	IncludeCodingStandards bool
	// IncludeSecurityPolicy controls whether security instructions are
	// generated.
	IncludeSecurityPolicy bool

	// Options carries language-specific answers, such as a Go module path or a
	// Maven group id, keyed by constants the owning plugin declares. The core
	// moves these without interpreting them; that is what keeps this package
	// free of language knowledge.
	Options map[string]string
}

// Default returns a ProjectConfig pre-populated with the opinionated defaults
// of the factory: every professional feature enabled, MIT licence, trunk-based
// branch name. Callers layer flags and prompt answers on top, so the cheapest
// path produces the most complete repository.
func Default() ProjectConfig {
	return ProjectConfig{
		License:                "MIT",
		DefaultBranch:          DefaultBranch,
		Options:                map[string]string{},
		InitializeGit:          true,
		IncludeClaude:          true,
		IncludeClaudeAgents:    true,
		IncludeClaudeWorkflows: true,
		IncludeGitHubActions:   true,
		IncludeDocs:            true,
		IncludeTests:           true,
		IncludePRTemplate:      true,
		IncludeCodingStandards: true,
		IncludeSecurityPolicy:  true,
	}
}

// Option returns the language-specific option stored under key, or fallback
// when it is absent or blank.
func (c ProjectConfig) Option(key, fallback string) string {
	if v, ok := c.Options[key]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

// SetOption records a language-specific option, allocating the map on demand
// so a zero ProjectConfig stays usable.
func (c *ProjectConfig) SetOption(key, value string) {
	if c.Options == nil {
		c.Options = map[string]string{}
	}
	c.Options[key] = value
}

// OptionKeys returns the option keys in sorted order, which keeps rendered
// output and test assertions deterministic.
func (c ProjectConfig) OptionKeys() []string {
	keys := make([]string, 0, len(c.Options))
	for k := range c.Options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ResolvedOutputDirectory returns the absolute directory the repository will
// be created in. An empty OutputDirectory resolves to ProjectName under the
// working directory.
//
// Call Validate first: this method resolves a path, it does not vet one.
func (c ProjectConfig) ResolvedOutputDirectory() (string, error) {
	dir := strings.TrimSpace(c.OutputDirectory)
	if dir == "" {
		dir = c.ProjectName
	}
	return filepath.Abs(dir)
}

// HasLicense reports whether a licence file should be generated.
func (c ProjectConfig) HasLicense() bool {
	l := strings.TrimSpace(strings.ToLower(c.License))
	return l != "" && l != LicenseUnlicensed
}
