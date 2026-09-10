// Package lang holds the language plugins shipped with the factory and the
// default registry they are registered in.
//
// Adding a language means adding one file to this package that declares a
// Definition and registers it in an init function. Nothing in the core
// (spec, plugin, render, cli) changes.
package lang

import (
	"fmt"
	"strings"

	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// defaultRegistry is populated by the init functions in this package.
var defaultRegistry = plugin.NewRegistry()

// Registry returns the registry containing every built-in language plugin.
func Registry() *plugin.Registry { return defaultRegistry }

// register adds a definition to the default registry. It panics on a
// mis-declared plugin, which surfaces as a start-up failure in tests rather
// than a confusing runtime error later.
func register(d *Definition) {
	defaultRegistry.MustRegister(d)
}

// Definition is the declarative implementation of plugin.Language used by
// every built-in plugin. It keeps a language plugin to data plus a couple of
// small functions, so the boilerplate of a new language stays near zero.
type Definition struct {
	// Meta is the static metadata exposed to the CLI.
	Meta plugin.Descriptor

	// RequiredOptions lists Spec option keys the language cannot generate
	// without, for example a module path.
	RequiredOptions []OptionSpec

	// Instruct returns the CLAUDE.md fragments for this language. It must
	// not be nil.
	Instruct func(s spec.Spec) plugin.Instructions
}

// OptionSpec describes a language-specific answer collected from flags or
// prompts and stored in Spec.Options.
type OptionSpec struct {
	Key      string
	Prompt   string
	Help     string
	Default  func(s spec.Spec) string
	Required bool
}

// DefaultValue returns the computed default for an option, or the empty
// string when the option has none.
func (o OptionSpec) DefaultValue(s spec.Spec) string {
	if o.Default == nil {
		return ""
	}
	return o.Default(s)
}

// Descriptor implements plugin.Language.
func (d *Definition) Descriptor() plugin.Descriptor { return d.Meta }

// Instructions implements plugin.Language.
func (d *Definition) Instructions(s spec.Spec) plugin.Instructions {
	if d.Instruct == nil {
		return plugin.Instructions{}
	}
	return d.Instruct(s)
}

// Files implements plugin.Language.
//
// Repository generation is deliberately not implemented yet: this milestone
// establishes the plugin contract and the CLI around it. Every built-in
// plugin reports the same honest error until the generator lands.
func (d *Definition) Files(spec.Spec) ([]plugin.FileSpec, error) {
	return nil, fmt.Errorf("%s: %w", d.Meta.ID, plugin.ErrNotImplemented)
}

// Validate implements plugin.Language.
func (d *Definition) Validate(s spec.Spec) error {
	if d.Meta.Status != plugin.StatusStable {
		return fmt.Errorf("language %q is planned but not available yet", d.Meta.ID)
	}
	if pt := strings.TrimSpace(s.ProjectType); pt != "" && !d.Meta.HasProjectType(pt) {
		return fmt.Errorf("unknown project type %q for %s (available: %s)",
			pt, d.Meta.ID, strings.Join(d.Meta.ProjectTypeIDs(), ", "))
	}
	for _, opt := range d.RequiredOptions {
		if !opt.Required {
			continue
		}
		if strings.TrimSpace(s.Option(opt.Key, "")) == "" {
			return fmt.Errorf("%s requires the %q option (%s)", d.Meta.ID, opt.Key, opt.Prompt)
		}
	}
	return nil
}

// Options returns the option specs of a language, or nil when the language
// is not a Definition. It lets the CLI collect language-specific answers
// without knowing anything about the language itself.
func Options(l plugin.Language) []OptionSpec {
	if d, ok := l.(*Definition); ok {
		return d.RequiredOptions
	}
	return nil
}
