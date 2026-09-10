// Package plugin defines the extension contract of the factory.
//
// The core generator knows nothing about Node.js, Python, Go or Java. It
// knows only about the Language interface declared here. Adding support for
// a new language or framework therefore means writing a new implementation
// of Language and registering it: no change to the generator, the CLI or the
// CLAUDE.md assembler is required.
package plugin

import (
	"errors"
	"io/fs"

	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// ErrNotImplemented is returned by a Language whose file generation has not
// been built yet. The CLI reports it as a clear "not available" message
// rather than a crash.
var ErrNotImplemented = errors.New("plugin: generation not implemented yet")

// Status describes how far a language plugin has been taken.
type Status string

const (
	// StatusStable means the plugin is fully wired and selectable.
	StatusStable Status = "stable"
	// StatusPlanned means the plugin is a placeholder announcing intent: it
	// appears in languages output but cannot be selected for generation.
	StatusPlanned Status = "planned"
)

// ProjectType is a variant within a language, such as cli, service or
// library. Plugins own their own project types.
type ProjectType struct {
	ID          string
	DisplayName string
	Summary     string
}

// Descriptor is the static, spec-independent metadata of a language plugin.
// The CLI uses it for listing, lookup and prompting.
type Descriptor struct {
	ID                 string
	DisplayName        string
	Summary            string
	Aliases            []string
	Status             Status
	ProjectTypes       []ProjectType
	DefaultProjectType string
}

// Command is a named command a generated repository supports, for example
// test -> go test ./... . These flow into both CLAUDE.md and CI so the two
// can never disagree about how the project is built.
type Command struct {
	Name        string
	Run         string
	Description string
}

// Instructions is the language-specific contribution to a generated
// CLAUDE.md. The assembler owns the universal sections and the ordering; a
// plugin only fills in what is genuinely language-specific.
type Instructions struct {
	// Toolchain describes required runtimes, versions and package managers.
	Toolchain string
	// Standards holds coding standards and idioms for the language.
	Standards string
	// Testing describes the test layout, runner and coverage expectations.
	Testing string
	// Security lists language-specific security rules and scanners.
	Security string
	// Architecture describes the expected source layout.
	Architecture string
	// Commands are the canonical build, test and lint invocations.
	Commands []Command
}

// FileSpec describes a single file a plugin wants written into a generated
// repository. Path is always slash-separated and relative to the repository
// root; the writer translates it to the host separator.
type FileSpec struct {
	Path     string
	Template string
	Data     map[string]any
	Mode     fs.FileMode
}

// Language is the contract every language or framework plugin implements.
//
// Implementations must be safe for concurrent use and must not mutate the
// Spec they are handed.
type Language interface {
	// Descriptor returns static metadata about the plugin.
	Descriptor() Descriptor
	// Instructions returns the CLAUDE.md fragments for this language.
	Instructions(s spec.Spec) Instructions
	// Files returns the files to generate. Plugins that are not finished
	// return ErrNotImplemented.
	Files(s spec.Spec) ([]FileSpec, error)
	// Validate checks language-specific requirements of a Spec, such as a
	// mandatory module path. It returns nil when the Spec is acceptable.
	Validate(s spec.Spec) error
}

// ProjectTypeIDs returns the project type identifiers of a descriptor in
// declaration order.
func (d Descriptor) ProjectTypeIDs() []string {
	ids := make([]string, 0, len(d.ProjectTypes))
	for _, pt := range d.ProjectTypes {
		ids = append(ids, pt.ID)
	}
	return ids
}

// HasProjectType reports whether id is a project type of this descriptor.
func (d Descriptor) HasProjectType(id string) bool {
	for _, pt := range d.ProjectTypes {
		if pt.ID == id {
			return true
		}
	}
	return false
}
