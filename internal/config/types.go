package config

import "strings"

// Language identifies the language plugin that will shape a repository.
//
// It is a named string type rather than a closed enum on purpose. Making it a
// distinct type stops a bare string being passed where a language is meant,
// which is the type safety worth having; freezing the permitted values here
// would instead put language names in the core, and ADR 0001 forbids that
// because it is what makes adding a language a core edit. Which languages are
// valid is therefore answered by a Catalog, supplied by the caller and backed
// by the plugin registry.
type Language string

// String implements fmt.Stringer.
func (l Language) String() string { return string(l) }

// ParseLanguage normalises user input into a Language. It only canonicalises
// case and whitespace; it does not decide whether the language exists, which
// is the Catalog's job.
func ParseLanguage(s string) Language {
	return Language(strings.ToLower(strings.TrimSpace(s)))
}

// ProjectType is the shape of the project being generated, independent of
// language: a Go API and a Python API are both APIs.
//
// Unlike Language this is a closed enum, and safely so — it names no language,
// so adding a language still requires no change here. It gives every plugin a
// shared vocabulary instead of each inventing its own, which is what lets the
// factory reason about project shape at all.
type ProjectType string

const (
	// ProjectTypeAPI is a service exposing a network API.
	ProjectTypeAPI ProjectType = "api"
	// ProjectTypeCLI is a command line tool.
	ProjectTypeCLI ProjectType = "cli"
	// ProjectTypeLibrary is a reusable package with a public API and no
	// entry point of its own.
	ProjectTypeLibrary ProjectType = "library"
	// ProjectTypeWorker is a background processor: a queue consumer, a
	// scheduled job, or a data pipeline.
	ProjectTypeWorker ProjectType = "worker"
)

// allProjectTypes is the canonical vocabulary, in the order it is presented
// to users.
var allProjectTypes = []ProjectType{
	ProjectTypeAPI,
	ProjectTypeCLI,
	ProjectTypeLibrary,
	ProjectTypeWorker,
}

// ProjectTypes returns every project type the factory recognises, in a stable
// order. The returned slice is a copy, so a caller cannot mutate the
// vocabulary.
func ProjectTypes() []ProjectType {
	out := make([]ProjectType, len(allProjectTypes))
	copy(out, allProjectTypes)
	return out
}

// String implements fmt.Stringer.
func (p ProjectType) String() string { return string(p) }

// Valid reports whether p is one of the recognised project types.
func (p ProjectType) Valid() bool {
	for _, known := range allProjectTypes {
		if p == known {
			return true
		}
	}
	return false
}

// ParseProjectType normalises and validates user input in one step. It
// returns ErrUnsupportedProjectType for anything outside the vocabulary.
func ParseProjectType(s string) (ProjectType, error) {
	p := ProjectType(strings.ToLower(strings.TrimSpace(s)))
	if !p.Valid() {
		return "", newFieldError(ErrUnsupportedProjectType, "ProjectType", s,
			"must be one of "+JoinProjectTypes(allProjectTypes, ", "))
	}
	return p, nil
}

// JoinProjectTypes renders project types for an error message or a prompt.
func JoinProjectTypes(types []ProjectType, sep string) string {
	return joinStringers(len(types), sep, func(i int) string { return string(types[i]) })
}

// joinStringers renders a slice of named string types without each caller
// repeating the same conversion loop.
func joinStringers(n int, sep string, at func(int) string) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = at(i)
	}
	return strings.Join(parts, sep)
}

// PackageManager identifies the dependency tool a generated repository uses.
//
// Like Language this is a named string type without a frozen value set: which
// package managers exist is ecosystem knowledge, and ecosystem knowledge lives
// in the language plugin, not in the core. The Catalog answers which ones a
// given language accepts.
type PackageManager string

// String implements fmt.Stringer.
func (m PackageManager) String() string { return string(m) }

// ParsePackageManager normalises user input into a PackageManager without
// deciding whether the language supports it.
func ParsePackageManager(s string) PackageManager {
	return PackageManager(strings.ToLower(strings.TrimSpace(s)))
}
