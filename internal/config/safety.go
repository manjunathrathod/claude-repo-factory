package config

import (
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MaxProjectNameLength bounds a project name. The limit is well under the 255
// bytes a filesystem component usually allows, because the name also becomes
// part of longer generated paths.
const MaxProjectNameLength = 64

// projectNamePattern is deliberately narrow. A project name becomes a
// directory name, part of a Git remote URL and part of generated identifiers,
// so it admits only characters that are unambiguous in all three.
var projectNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// windowsReservedNames are device names that cannot be used as a file or
// directory name on Windows, with or without an extension. Creating one fails
// in confusing ways, so they are rejected on every platform to keep generated
// repositories portable.
// This is exactly the set Microsoft documents as reserved. COM0 and LPT0 are
// deliberately absent: they are not reserved, and rejecting them would refuse
// a legitimate name for no reason.
var windowsReservedNames = map[string]struct{}{
	"con": {}, "prn": {}, "aux": {}, "nul": {},
	"com1": {}, "com2": {}, "com3": {}, "com4": {}, "com5": {},
	"com6": {}, "com7": {}, "com8": {}, "com9": {},
	"lpt1": {}, "lpt2": {}, "lpt3": {}, "lpt4": {}, "lpt5": {},
	"lpt6": {}, "lpt7": {}, "lpt8": {}, "lpt9": {},
}

// ValidateProjectName reports whether name is safe to use as the directory
// name and identifier of a generated repository.
//
// The rules reject, in order: an empty name, an over-long name, a name
// containing a character outside the permitted set (which excludes every path
// separator, so a name can never contribute a path segment), a name that
// contains "..", a name ending in a dot, and a Windows reserved device name.
func ValidateProjectName(name string) error {
	const field = "ProjectName"

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return newFieldError(ErrMissingField, field, "", "must not be empty")
	}
	if trimmed != name {
		return newFieldError(ErrUnsafeName, field, name,
			"must not begin or end with whitespace")
	}
	if len(name) > MaxProjectNameLength {
		return newFieldError(ErrUnsafeName, field, name,
			"must be "+strconv.Itoa(MaxProjectNameLength)+" characters or fewer")
	}
	if !projectNamePattern.MatchString(name) {
		return newFieldError(ErrUnsafeName, field, name,
			"must start with a letter or digit and contain only letters, digits, '.', '_' or '-'")
	}
	// Defence in depth: the pattern above already excludes separators, so a
	// name cannot form a path segment. Rejecting ".." as well means no name
	// can ever read as a traversal component if the pattern is later widened.
	if strings.Contains(name, "..") {
		return newFieldError(ErrUnsafeName, field, name,
			"must not contain '..'")
	}
	if strings.HasSuffix(name, ".") {
		return newFieldError(ErrUnsafeName, field, name,
			"must not end with '.'")
	}
	if isWindowsReserved(name) {
		return newFieldError(ErrUnsafeName, field, name,
			"is a reserved device name on Windows")
	}
	return nil
}

// isWindowsReserved reports whether name matches a Windows device name,
// ignoring case and any extension: both "NUL" and "nul.txt" are reserved.
func isWindowsReserved(name string) bool {
	base := name
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	_, reserved := windowsReservedNames[strings.ToLower(base)]
	return reserved
}

// ValidateOutputDirectory reports whether dir is safe to create a repository
// in. An empty dir is valid and means "use the project name".
//
// The check is deliberately performed on the cleaned, slash-normalised form:
// comparing raw strings would miss "a/../../etc" and would differ between
// Windows and POSIX.
func ValidateOutputDirectory(dir string) error {
	const field = "OutputDirectory"

	if strings.TrimSpace(dir) == "" {
		return nil
	}
	if strings.ContainsRune(dir, 0) {
		return newFieldError(ErrInvalidValue, field, dir,
			"must not contain a NUL byte")
	}

	// Check the raw segments, not the cleaned path. path.Clean resolves ".."
	// away — and for an absolute path it discards a ".." that would escape the
	// root, so "/srv/../../etc" cleans to "/etc" and the traversal disappears
	// before it can be detected. The policy is therefore strict: no ".."
	// segment is accepted anywhere, even one that would stay inside the base.
	// A scaffolding target has no legitimate need for one, and a rule with no
	// exceptions is the only kind that stays correct as callers change.
	normalised := filepath.ToSlash(dir)
	for _, segment := range strings.Split(normalised, "/") {
		if segment == ".." {
			return newFieldError(ErrPathTraversal, field, dir,
				"must not contain a '..' segment")
		}
	}

	cleaned := path.Clean(normalised)
	if isFilesystemRoot(cleaned) {
		return newFieldError(ErrInvalidValue, field, dir,
			"must not be a filesystem root")
	}
	return nil
}

// isFilesystemRoot reports whether a cleaned, slash-separated path is a root
// such as "/", "C:" or "C:/". Generating a repository directly into a root is
// always a mistake and usually a typo.
func isFilesystemRoot(cleaned string) bool {
	if cleaned == "/" {
		return true
	}
	// A Windows drive root: exactly a letter, a colon, and nothing else once
	// any trailing separator is removed.
	trimmed := strings.TrimSuffix(cleaned, "/")
	return len(trimmed) == 2 && trimmed[1] == ':' && isASCIILetter(trimmed[0])
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
