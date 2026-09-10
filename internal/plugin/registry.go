package plugin

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry holds the known language plugins and resolves user input
// (identifiers or aliases) to them. The zero value is not usable; call
// NewRegistry.
type Registry struct {
	mu        sync.RWMutex
	languages map[string]Language
	aliases   map[string]string
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		languages: map[string]Language{},
		aliases:   map[string]string{},
	}
}

// Register adds a language plugin. It fails on an empty or duplicate
// identifier, or on an alias already claimed by another plugin, so a
// mis-wired plugin is caught at start-up rather than at generation time.
func (r *Registry) Register(l Language) error {
	if l == nil {
		return fmt.Errorf("registry: cannot register a nil language")
	}
	d := l.Descriptor()
	id := normalise(d.ID)
	if id == "" {
		return fmt.Errorf("registry: language %q has an empty id", d.DisplayName)
	}
	if d.DefaultProjectType != "" && !d.HasProjectType(d.DefaultProjectType) {
		return fmt.Errorf("registry: language %q declares unknown default project type %q", id, d.DefaultProjectType)
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
		if _, clash := r.languages[a]; clash {
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

	if l, ok := r.languages[key]; ok {
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
	return nil, fmt.Errorf("unknown language %q (available: %s)", name, strings.Join(r.IDs(), ", "))
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

// Descriptors returns the descriptors of every registered language, ordered
// by identifier.
func (r *Registry) Descriptors() []Descriptor {
	langs := r.List()
	out := make([]Descriptor, 0, len(langs))
	for _, l := range langs {
		out = append(out, l.Descriptor())
	}
	return out
}

// IDs returns the registered language identifiers in sorted order.
func (r *Registry) IDs() []string {
	langs := r.List()
	out := make([]string, 0, len(langs))
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
