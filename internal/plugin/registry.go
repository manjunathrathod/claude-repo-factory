package plugin

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
)

// Registry implements config.Catalog, which is what lets validation ask this
// build what it supports instead of consulting a list duplicated in the core.
var _ config.Catalog = (*Registry)(nil)

// Registry holds the known language plugins and resolves user input
// (identifiers or aliases) to them. The zero value is not usable; call
// NewRegistry.
type Registry struct {
	mu        sync.RWMutex
	languages map[config.Language]Language
	aliases   map[string]config.Language
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		languages: map[config.Language]Language{},
		aliases:   map[string]config.Language{},
	}
}

// Register adds a language plugin.
//
// Every consistency rule a plugin must satisfy is enforced here, so a
// mis-declared plugin fails at start-up rather than part-way through
// generating somebody's repository.
func (r *Registry) Register(l Language) error {
	if l == nil {
		return fmt.Errorf("registry: cannot register a nil language")
	}
	d := l.Descriptor()
	id := config.ParseLanguage(string(d.ID))
	if id == "" {
		return fmt.Errorf("registry: language %q has an empty id", d.DisplayName)
	}
	if err := validateDescriptor(id, d); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.languages[id]; exists {
		return fmt.Errorf("registry: language %q is already registered", id)
	}
	for _, alias := range d.Aliases {
		a := normalise(alias)
		if a == "" {
			continue
		}
		if owner, taken := r.aliases[a]; taken && owner != id {
			return fmt.Errorf("registry: alias %q for language %q is already used by %q", a, id, owner)
		}
		if _, clash := r.languages[config.Language(a)]; clash {
			return fmt.Errorf("registry: alias %q for language %q collides with a language id", a, id)
		}
	}

	r.languages[id] = l
	for _, alias := range d.Aliases {
		if a := normalise(alias); a != "" {
			r.aliases[a] = id
		}
	}
	return nil
}

// validateDescriptor checks the internal consistency of a descriptor. Planned
// plugins are held to a lighter standard because they exist only to reserve a
// name and announce intent.
func validateDescriptor(id config.Language, d Descriptor) error {
	for _, pt := range d.ProjectTypes {
		if !pt.ID.Valid() {
			return fmt.Errorf("registry: language %q declares project type %q, which is not in the factory vocabulary (%s)",
				id, pt.ID, config.JoinProjectTypes(config.ProjectTypes(), ", "))
		}
	}
	if d.DefaultProjectType != "" && !d.HasProjectType(d.DefaultProjectType) {
		return fmt.Errorf("registry: language %q declares unknown default project type %q", id, d.DefaultProjectType)
	}
	if d.DefaultPackageManager != "" && !d.HasPackageManager(d.DefaultPackageManager) {
		return fmt.Errorf("registry: language %q declares unknown default package manager %q", id, d.DefaultPackageManager)
	}
	if d.Status != StatusStable {
		return nil
	}
	if len(d.ProjectTypes) == 0 {
		return fmt.Errorf("registry: stable language %q declares no project types", id)
	}
	if len(d.PackageManagers) == 0 {
		return fmt.Errorf("registry: stable language %q declares no package managers", id)
	}
	if d.DefaultProjectType == "" {
		return fmt.Errorf("registry: stable language %q declares no default project type", id)
	}
	if d.DefaultPackageManager == "" {
		return fmt.Errorf("registry: stable language %q declares no default package manager", id)
	}
	return nil
}

// MustRegister is Register for package initialisation, where a failure is a
// programming error rather than a runtime condition.
func (r *Registry) MustRegister(l Language) {
	if err := r.Register(l); err != nil {
		panic(err)
	}
}

// Lookup resolves an identifier or alias, case-insensitively.
func (r *Registry) Lookup(name string) (Language, bool) {
	key := normalise(name)
	if key == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if l, ok := r.languages[config.Language(key)]; ok {
		return l, true
	}
	if id, ok := r.aliases[key]; ok {
		l, ok := r.languages[id]
		return l, ok
	}
	return nil, false
}

// Get resolves a language or returns an error listing the valid choices.
func (r *Registry) Get(name string) (Language, error) {
	if l, ok := r.Lookup(name); ok {
		return l, nil
	}
	return nil, fmt.Errorf("unknown language %q (available: %s)",
		name, config.JoinLanguages(r.Languages(), ", "))
}

// Resolve implements config.Catalog. It maps an id or alias to the canonical
// language identifier.
func (r *Registry) Resolve(name string) (config.Language, bool) {
	l, ok := r.Lookup(name)
	if !ok {
		return "", false
	}
	return l.Descriptor().ID, true
}

// Languages implements config.Catalog. Only stable plugins are reported: a
// planned language is announced in listings but must never pass validation.
func (r *Registry) Languages() []config.Language {
	// Sized from Available rather than from r.languages: reading the map
	// length here would touch shared state without holding the lock, and
	// Available takes it for us.
	available := r.Available()
	out := make([]config.Language, 0, len(available))
	for _, l := range available {
		out = append(out, l.Descriptor().ID)
	}
	return out
}

// ProjectTypes implements config.Catalog.
func (r *Registry) ProjectTypes(l config.Language) []config.ProjectType {
	found, ok := r.Lookup(string(l))
	if !ok {
		return nil
	}
	return found.Descriptor().ProjectTypeIDs()
}

// PackageManagers implements config.Catalog.
func (r *Registry) PackageManagers(l config.Language) []config.PackageManager {
	found, ok := r.Lookup(string(l))
	if !ok {
		return nil
	}
	d := found.Descriptor()
	out := make([]config.PackageManager, len(d.PackageManagers))
	copy(out, d.PackageManagers)
	return out
}

// List returns every registered language ordered by identifier.
func (r *Registry) List() []Language {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Language, 0, len(r.languages))
	for _, l := range r.languages {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Descriptor().ID < out[j].Descriptor().ID
	})
	return out
}

// Descriptors returns the descriptors of every registered language, ordered by
// identifier.
func (r *Registry) Descriptors() []Descriptor {
	langs := r.List()
	out := make([]Descriptor, 0, len(langs))
	for _, l := range langs {
		out = append(out, l.Descriptor())
	}
	return out
}

// IDs returns the registered language identifiers in sorted order, including
// planned ones. Use Languages for the set validation should accept.
func (r *Registry) IDs() []config.Language {
	langs := r.List()
	out := make([]config.Language, 0, len(langs))
	for _, l := range langs {
		out = append(out, l.Descriptor().ID)
	}
	return out
}

// Available returns the languages that can currently be generated.
func (r *Registry) Available() []Language {
	var out []Language
	for _, l := range r.List() {
		if l.Descriptor().Status == StatusStable {
			out = append(out, l)
		}
	}
	return out
}

func normalise(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
