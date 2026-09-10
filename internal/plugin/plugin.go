// Package plugin defines the extension contract of the factory.
//
// The core generator knows nothing about Node.js, Python, Go or Java. It knows
// only about the Language interface declared here. Adding support for a new
// language or framework therefore means writing a new implementation of
// Language and registering it: no change to the generator, the CLI or the
// CLAUDE.md assembler is required.
//
// This package also supplies the answer to "what does this build support":
// *Registry implements config.Catalog, so validation is performed against the
// plugins actually registered rather than a list duplicated in the core.
package plugin

import (
	"errors"
	"io/fs"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
)

// ErrNotImplemented is returned by a Language whose file generation has not
// been built yet. The CLI reports it as a clear "not available" message rather
// than a crash.
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

// ProjectType is one project type as a plugin presents it. The ID comes from
// the closed factory vocabulary; the display text belongs to the plugin, so
// two languages can describe the same shape in their own terms.
type ProjectType struct {
	ID          config.ProjectType
	DisplayName string
	Summary     string
}

// Descriptor is the static, spec-independent metadata of a language plugin.
// The CLI uses it for listing, lookup and prompting.
type Descriptor struct {
	// ID is the canonical language identifier, such as "nodejs".
	ID config.Language
	// DisplayName is the human-facing name, such as "Node.js / TypeScript".
	DisplayName string
	// Summary is a one-line description of the toolchain.
	Summary string
	// Aliases are alternative spellings accepted as input, such as "node".
	Aliases []string
	// Status reports whether the plugin can currently be generated.
	Status Status
	// ProjectTypes are the shapes this language supports. A plugin may narrow
	// the factory vocabulary; it may never widen it.
	ProjectTypes []ProjectType
	// DefaultProjectType must be one of ProjectTypes.
	DefaultProjectType config.ProjectType
	// PackageManagers are the dependency tools this language supports, in the
	// order they should be offered.
	PackageManagers []config.PackageManager
	// DefaultPackageManager must be one of PackageManagers.
	DefaultPackageManager config.PackageManager
}

// Command is a named command a generated repository supports, for example
// test -> go test ./... . These flow into both CLAUDE.md and CI so the two can
// never disagree about how the project is built.
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
// ProjectConfig they are handed.
type Language interface {
	// Descriptor returns static metadata about the plugin.
	Descriptor() Descriptor
	// Instructions returns the CLAUDE.md fragments for this language.
	Instructions(c config.ProjectConfig) Instructions
	// Files returns the files to generate. Plugins that are not finished
	// return ErrNotImplemented.
	Files(c config.ProjectConfig) ([]FileSpec, error)
	// Validate checks language-specific requirements, such as a mandatory
	// module path. Language-agnostic rules are config.ProjectConfig.Validate.
	Validate(c config.ProjectConfig) error
}

// ProjectTypeIDs returns the project type identifiers of a descriptor in
// declaration order.
func (d Descriptor) ProjectTypeIDs() []config.ProjectType {
	ids := make([]config.ProjectType, 0, len(d.ProjectTypes))
	for _, pt := range d.ProjectTypes {
		ids = append(ids, pt.ID)
	}
	return ids
}

// HasProjectType reports whether id is a project type of this descriptor.
func (d Descriptor) HasProjectType(id config.ProjectType) bool {
	for _, pt := range d.ProjectTypes {
		if pt.ID == id {
			return true
		}
	}
	return false
}

// HasPackageManager reports whether m is a package manager of this descriptor.
func (d Descriptor) HasPackageManager(m config.PackageManager) bool {
	for _, known := range d.PackageManagers {
		if known == m {
			return true
		}
	}
	return false
}

// ProjectTypeDisplayName returns the plugin's human-facing name for a project
// type, such as "API" for "api". It falls back to the identifier so a caller
// always has something to print, even for a type the plugin does not declare.
func (d Descriptor) ProjectTypeDisplayName(id config.ProjectType) string {
	for _, pt := range d.ProjectTypes {
		if pt.ID == id {
			return pt.DisplayName
		}
	}
	return string(id)
}
