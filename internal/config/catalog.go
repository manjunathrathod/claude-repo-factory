package config

// Catalog reports what this build of the factory actually supports.
//
// It exists so that validation can reject an unsupported language, project
// type or package manager without this package ever naming one. The plugin
// registry implements it, which makes the registered plugin set the single
// source of truth: a frozen list here would be a second source that drifts the
// first time a plugin is added or renamed. See ADR 0001 and ADR 0003.
//
// Implementations must be safe for concurrent use and must return stable
// ordering, since the results are rendered into error messages and prompts.
type Catalog interface {
	// Languages returns every language that can currently be generated,
	// ordered by identifier.
	Languages() []Language

	// Resolve maps user input — a canonical id or an alias, in any case — to
	// the canonical Language. It reports false for anything unknown.
	Resolve(name string) (Language, bool)

	// ProjectTypes returns the project types the language supports, in the
	// order they should be offered. It returns nil for an unknown language.
	ProjectTypes(l Language) []ProjectType

	// PackageManagers returns the package managers the language supports, in
	// the order they should be offered. It returns nil for an unknown
	// language.
	PackageManagers(l Language) []PackageManager
}

// SupportsLanguage reports whether the catalog knows the canonical language.
func SupportsLanguage(c Catalog, l Language) bool {
	for _, known := range c.Languages() {
		if known == l {
			return true
		}
	}
	return false
}

// JoinLanguages renders languages for an error message or a prompt.
func JoinLanguages(langs []Language, sep string) string {
	return joinStringers(len(langs), sep, func(i int) string { return string(langs[i]) })
}

// JoinPackageManagers renders package managers for an error message.
func JoinPackageManagers(managers []PackageManager, sep string) string {
	return joinStringers(len(managers), sep, func(i int) string { return string(managers[i]) })
}
