package config_test

import "github.com/manjunathrathod/claude-repo-factory/internal/config"

// fakeCatalog is a Catalog with a fixed, inspectable answer set. Using a fake
// rather than the real registry keeps these tests independent of which
// plugins happen to be registered, which is the whole point of the Catalog
// interface existing.
type fakeCatalog struct {
	languages []config.Language
	aliases   map[string]config.Language
	types     map[config.Language][]config.ProjectType
	managers  map[config.Language][]config.PackageManager
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{
		languages: []config.Language{"go", "python"},
		aliases:   map[string]config.Language{"golang": "go", "py": "python"},
		types: map[config.Language][]config.ProjectType{
			"go":     {config.ProjectTypeAPI, config.ProjectTypeCLI, config.ProjectTypeLibrary},
			"python": {config.ProjectTypeAPI, config.ProjectTypeLibrary, config.ProjectTypeWorker},
		},
		managers: map[config.Language][]config.PackageManager{
			"go":     {"gomod"},
			"python": {"uv", "pip"},
		},
	}
}

func (f *fakeCatalog) Languages() []config.Language { return f.languages }

func (f *fakeCatalog) Resolve(name string) (config.Language, bool) {
	l := config.ParseLanguage(name)
	for _, known := range f.languages {
		if known == l {
			return known, true
		}
	}
	if id, ok := f.aliases[string(l)]; ok {
		return id, true
	}
	return "", false
}

func (f *fakeCatalog) ProjectTypes(l config.Language) []config.ProjectType {
	return f.types[l]
}

func (f *fakeCatalog) PackageManagers(l config.Language) []config.PackageManager {
	return f.managers[l]
}

// validConfig returns a configuration that passes validation against
// newFakeCatalog, so each test can mutate exactly the one field under test.
func validConfig() config.ProjectConfig {
	c := config.Default()
	c.ProjectName = "widget"
	c.Description = "Widget control plane"
	c.Language = "go"
	c.ProjectType = config.ProjectTypeCLI
	c.PackageManager = "gomod"
	c.OutputDirectory = "widget"
	return c
}
