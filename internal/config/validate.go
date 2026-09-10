package config

import (
	"errors"
	"strings"
)

// branchUnsafeChars are the characters Git refuses in a branch name, plus
// whitespace. The full refname grammar is larger; this covers the mistakes a
// user actually makes at a prompt.
const branchUnsafeChars = " ~^:?*[\\"

// Validate reports whether the configuration is complete, safe and supported.
//
// It reports every problem at once rather than the first, so a user fixes one
// round of mistakes instead of rediscovering them one command at a time. Each
// returned error is a *FieldError wrapping one of the package sentinels, so
// callers can match a category with errors.Is or read the field name with
// errors.As.
//
// Support questions — is this language registered, does it offer this project
// type — are asked of the catalog rather than of a list held here, which is
// what keeps this package free of language knowledge. A nil catalog is an
// error rather than a skipped check: silently accepting any language would
// turn a validation call into a no-op.
//
// Validate never touches the filesystem. Whether the output directory exists
// or is empty is the writer's concern at generation time, not the model's.
func (c ProjectConfig) Validate(catalog Catalog) error {
	var errs []error

	if err := ValidateProjectName(c.ProjectName); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateOutputDirectory(c.OutputDirectory); err != nil {
		errs = append(errs, err)
	}
	if err := validateBranch(c.DefaultBranch); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateRemote(c.Remote); err != nil {
		errs = append(errs, err)
	}
	// Free-text fields reach the confirmation summary and the generated
	// README, so they may not carry characters that forge output lines.
	for _, field := range []struct{ name, value string }{
		{"Description", c.Description},
		{"Author", c.Author},
		{"License", c.License},
	} {
		if err := ValidateNoControlCharacters(field.name, field.value); err != nil {
			errs = append(errs, err)
		}
	}

	if catalog == nil {
		errs = append(errs, newFieldError(ErrInvalidValue, "Catalog", "",
			"a catalog is required to validate the language, project type and package manager"))
		return errors.Join(errs...)
	}

	errs = append(errs, c.validateAgainst(catalog)...)
	return errors.Join(errs...)
}

// validateAgainst checks the three catalog-backed fields. It is separate from
// Validate so the nil-catalog guard above stays readable.
func (c ProjectConfig) validateAgainst(catalog Catalog) []error {
	var errs []error

	if strings.TrimSpace(string(c.Language)) == "" {
		errs = append(errs, newFieldError(ErrMissingField, "Language", "",
			"must be one of "+JoinLanguages(catalog.Languages(), ", ")))
		// Every remaining check is language-relative, so stop here rather
		// than emitting three confusing follow-on errors.
		return errs
	}
	if !SupportsLanguage(catalog, c.Language) {
		errs = append(errs, newFieldError(ErrUnsupportedLanguage, "Language", string(c.Language),
			"must be one of "+JoinLanguages(catalog.Languages(), ", ")))
		return errs
	}

	errs = append(errs, c.validateProjectType(catalog)...)
	errs = append(errs, c.validatePackageManager(catalog)...)
	return errs
}

func (c ProjectConfig) validateProjectType(catalog Catalog) []error {
	supported := catalog.ProjectTypes(c.Language)

	if strings.TrimSpace(string(c.ProjectType)) == "" {
		return []error{newFieldError(ErrMissingField, "ProjectType", "",
			"must be one of "+JoinProjectTypes(supported, ", "))}
	}
	// Checked against the factory vocabulary first so an unknown value is
	// reported as such even for a language that supports only a subset.
	if !c.ProjectType.Valid() {
		return []error{newFieldError(ErrUnsupportedProjectType, "ProjectType", string(c.ProjectType),
			"must be one of "+JoinProjectTypes(ProjectTypes(), ", "))}
	}
	for _, p := range supported {
		if p == c.ProjectType {
			return nil
		}
	}
	return []error{newFieldError(ErrUnsupportedProjectType, "ProjectType", string(c.ProjectType),
		string(c.Language)+" supports "+JoinProjectTypes(supported, ", "))}
}

func (c ProjectConfig) validatePackageManager(catalog Catalog) []error {
	supported := catalog.PackageManagers(c.Language)

	if strings.TrimSpace(string(c.PackageManager)) == "" {
		return []error{newFieldError(ErrMissingField, "PackageManager", "",
			"must be one of "+JoinPackageManagers(supported, ", "))}
	}
	for _, m := range supported {
		if m == c.PackageManager {
			return nil
		}
	}
	return []error{newFieldError(ErrUnsupportedPackageManager, "PackageManager", string(c.PackageManager),
		string(c.Language)+" supports "+JoinPackageManagers(supported, ", "))}
}

// validateBranch checks the initial branch name.
func validateBranch(branch string) error {
	const field = "DefaultBranch"

	if strings.TrimSpace(branch) == "" {
		return newFieldError(ErrMissingField, field, "", "must not be empty")
	}
	if strings.ContainsAny(branch, branchUnsafeChars) {
		return newFieldError(ErrInvalidValue, field, branch,
			"contains characters Git does not allow in a branch name")
	}
	if strings.HasPrefix(branch, "-") || strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return newFieldError(ErrInvalidValue, field, branch,
			"must not begin with '-' or '/', or end with '/'")
	}
	if strings.Contains(branch, "..") || strings.Contains(branch, "//") {
		return newFieldError(ErrInvalidValue, field, branch,
			"must not contain '..' or '//'")
	}
	return nil
}
