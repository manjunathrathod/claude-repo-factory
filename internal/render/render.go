// Package render wraps text/template with the helper functions and the
// strictness settings the factory relies on.
//
// It is intentionally independent of language plugins: plugins supply
// template text and data, the engine turns them into bytes. Keeping the two
// apart is what lets a new language ship without touching the renderer.
package render

import (
	"bytes"
	"fmt"
	"io"
	"text/template"
)

// Engine renders templates with a fixed function map. It is safe for
// concurrent use.
type Engine struct {
	funcs template.FuncMap
}

// Option customises an Engine.
type Option func(*Engine)

// WithFuncs adds or overrides template helper functions.
func WithFuncs(funcs template.FuncMap) Option {
	return func(e *Engine) {
		for name, fn := range funcs {
			e.funcs[name] = fn
		}
	}
}

// New returns an Engine with the standard helper functions applied.
func New(opts ...Option) *Engine {
	e := &Engine{funcs: defaultFuncs()}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Parse compiles a template. Missing map keys are an error rather than
// silently rendering as <no value>, so a template referring to data that
// does not exist fails loudly during generation.
func (e *Engine) Parse(name, text string) (*template.Template, error) {
	t, err := template.New(name).Option("missingkey=error").Funcs(e.funcs).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("render: parse %s: %w", name, err)
	}
	return t, nil
}

// Execute renders text with data and writes the result to w.
func (e *Engine) Execute(w io.Writer, name, text string, data any) error {
	t, err := e.Parse(name, text)
	if err != nil {
		return err
	}
	if err := t.Execute(w, data); err != nil {
		return fmt.Errorf("render: execute %s: %w", name, err)
	}
	return nil
}

// String renders text with data and returns the result.
func (e *Engine) String(name, text string, data any) (string, error) {
	var buf bytes.Buffer
	if err := e.Execute(&buf, name, text, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
