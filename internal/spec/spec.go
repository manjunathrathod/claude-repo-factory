// Package spec defines the resolved description of a repository that the
// factory is asked to create.
//
// A Spec is deliberately free of any generation logic: it is the single,
// language-agnostic value that flows from the CLI (flags and interactive
// prompts) into language plugins. Language plugins read a Spec and decide
// which files to emit and which instructions to contribute to CLAUDE.md;
// they never reach back into the CLI.
package spec

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// LicenseUnlicensed is the value used when a repository intentionally ships
// without a license file.
const LicenseUnlicensed = "none"

// DefaultBranch is the branch name used for freshly initialised repositories
// unless the caller overrides it.
const DefaultBranch = "main"

// namePattern restricts repository names to characters that are safe on every
// supported platform and valid in Git remote URLs.
var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Features toggles the optional building blocks of a generated repository.
// Every field defaults to true via Default; the CLI only ever turns things
// off, which keeps "professional by default" the cheapest path.
type Features struct {
	Git             bool
	ClaudeConfig    bool
	ClaudeAgents    bool
	ClaudeWorkflows bool
	GitHubActions   bool
	Docs            bool
	Tests           bool
	PRTemplate      bool
	CodingStandards bool
	SecurityPolicy  bool
}

// Spec is the fully resolved description of a repository to generate.
type Spec struct {
	// Identity.
	Name        string
	Description string
	Author      string
	License     string

	// Placement.
	TargetDir     string
	DefaultBranch string
	Remote        string

	// Language plugin selection.
	Language    string
	ProjectType string

	// Options carries language-specific answers (for example a Go module
	// path or a Java group id) without leaking those concepts into the core.
	// Plugins own the key namespace for their own language.
	Options map[string]string

	Features Features
}

// Default returns a Spec pre-populated with the opinionated defaults of the
// factory. Callers layer flags and prompt answers on top of it.
func Default() Spec {
	return Spec{
		License:       "MIT",
		DefaultBranch: DefaultBranch,
		Options:       map[string]string{},
		Features: Features{
			Git:             true,
			ClaudeConfig:    true,
			ClaudeAgents:    true,
			ClaudeWorkflows: true,
			GitHubActions:   true,
			Docs:            true,
			Tests:           true,
			PRTemplate:      true,
			CodingStandards: true,
			SecurityPolicy:  true,
		},
	}
}

// Option returns the language-specific option stored under key, or fallback
// when it is absent or empty.
func (s Spec) Option(key, fallback string) string {
	if v, ok := s.Options[key]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

// SetOption records a language-specific option, allocating the map on demand
// so a zero Spec stays usable.
func (s *Spec) SetOption(key, value string) {
	if s.Options == nil {
		s.Options = map[string]string{}
	}
	s.Options[key] = value
}

// OptionKeys returns the option keys in stable sorted order, which keeps
// rendered output and test assertions deterministic.
func (s Spec) OptionKeys() []string {
	keys := make([]string, 0, len(s.Options))
	for k := range s.Options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Path returns the absolute directory the repository will be created in.
func (s Spec) Path() (string, error) {
	dir := s.TargetDir
	if dir == "" {
		dir = s.Name
	}
	return filepath.Abs(dir)
}

// HasLicense reports whether a license file should be generated.
func (s Spec) HasLicense() bool {
	l := strings.TrimSpace(strings.ToLower(s.License))
	return l != "" && l != LicenseUnlicensed
}

// Validate checks the language-agnostic invariants of a Spec. Language
// plugins add their own validation on top; this function never knows about
// specific languages.
func (s Spec) Validate() error {
	var errs []error

	switch {
	case strings.TrimSpace(s.Name) == "":
		errs = append(errs, errors.New("name: must not be empty"))
	case len(s.Name) > 100:
		errs = append(errs, errors.New("name: must be 100 characters or fewer"))
	case !namePattern.MatchString(s.Name):
		errs = append(errs, fmt.Errorf("name: %q must start with a letter or digit and contain only letters, digits, '.', '_' or '-'", s.Name))
	}

	if strings.TrimSpace(s.Language) == "" {
		errs = append(errs, errors.New("language: must not be empty"))
	}

	if strings.TrimSpace(s.DefaultBranch) == "" {
		errs = append(errs, errors.New("default branch: must not be empty"))
	} else if strings.ContainsAny(s.DefaultBranch, " ~^:?*[\\") {
		errs = append(errs, fmt.Errorf("default branch: %q contains characters Git does not allow", s.DefaultBranch))
	}

	return errors.Join(errs...)
}
