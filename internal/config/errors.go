package config

import (
	"fmt"
	"strings"
)

// Validation failures are grouped under these sentinels so a caller can react
// to a category without matching on message text. Every error returned by
// Validate wraps exactly one of them.
var (
	// ErrMissingField is returned when a required field is empty.
	ErrMissingField = sentinel("required field is missing")
	// ErrUnsafeName is returned when a project name could not be used safely
	// as a directory name on every supported platform.
	ErrUnsafeName = sentinel("unsafe project name")
	// ErrPathTraversal is returned when an output directory escapes, or could
	// escape, the directory it is resolved against.
	ErrPathTraversal = sentinel("path traversal")
	// ErrUnsupportedLanguage is returned for a language no plugin provides.
	ErrUnsupportedLanguage = sentinel("unsupported language")
	// ErrUnsupportedProjectType is returned for a project type the language
	// does not offer.
	ErrUnsupportedProjectType = sentinel("unsupported project type")
	// ErrUnsupportedPackageManager is returned for a package manager the
	// language does not support.
	ErrUnsupportedPackageManager = sentinel("unsupported package manager")
	// ErrInvalidValue is returned for a field that is malformed in a way none
	// of the more specific sentinels describes.
	ErrInvalidValue = sentinel("invalid value")
)

// sentinelError is a distinct type so that the package sentinels cannot be
// accidentally matched by an unrelated error carrying the same text.
type sentinelError string

func sentinel(s string) error { return sentinelError(s) }

func (e sentinelError) Error() string { return string(e) }

// FieldError reports a single invalid field. It carries the field name and
// the offending value so a CLI can point the user at the flag to fix, and it
// wraps a sentinel so callers can switch on the category.
type FieldError struct {
	// Field is the ProjectConfig field name, for example "ProjectName".
	Field string
	// Value is the value that was rejected. It is quoted when rendered.
	Value string
	// Reason explains what is wrong, in a form suitable for a user.
	Reason string
	// Kind is the sentinel this failure belongs to.
	Kind error
}

// Error implements error.
func (e *FieldError) Error() string {
	var b strings.Builder
	b.WriteString(e.Field)
	b.WriteString(": ")
	if e.Value != "" {
		fmt.Fprintf(&b, "%q: ", e.Value)
	}
	b.WriteString(e.Reason)
	return b.String()
}

// Unwrap returns the sentinel, so errors.Is(err, ErrUnsafeName) works through
// the joined error that Validate returns.
func (e *FieldError) Unwrap() error { return e.Kind }

// newFieldError builds a FieldError. It exists so the validation rules read
// as a list of conditions rather than a list of struct literals.
func newFieldError(kind error, field, value, reason string) *FieldError {
	return &FieldError{Field: field, Value: value, Reason: reason, Kind: kind}
}
