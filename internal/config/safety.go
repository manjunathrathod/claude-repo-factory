package config

import (
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"
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

// toSlash normalises both separators to "/" on every platform.
//
// filepath.ToSlash deliberately is not used: it converts only the host
// separator, so on Linux it leaves backslashes untouched and `..\..\etc`
// would sail past the traversal check that the same input fails on Windows.
// Validation must not depend on where it runs — a configuration is either
// safe or it is not — so a backslash is treated as a separator everywhere.
// The cost is that a directory name containing a literal backslash, which is
// legal on Unix, cannot be used; that is the right trade for a tool whose
// output is meant to be portable.
func toSlash(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
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
	if err := ValidateNoControlCharacters(field, dir); err != nil {
		return err
	}

	// Check the raw segments, not the cleaned path. path.Clean resolves ".."
	// away — and for an absolute path it discards a ".." that would escape the
	// root, so "/srv/../../etc" cleans to "/etc" and the traversal disappears
	// before it can be detected. The policy is therefore strict: no ".."
	// segment is accepted anywhere, even one that would stay inside the base.
	// A scaffolding target has no legitimate need for one, and a rule with no
	// exceptions is the only kind that stays correct as callers change.
	normalised := toSlash(dir)
	for _, segment := range strings.Split(normalised, "/") {
		if segment == ".." {
			return newFieldError(ErrPathTraversal, field, dir,
				"must not contain a '..' segment")
		}
	}

	// A leading "//" is a UNC share (\\server\share) or, on Windows, one of the
	// device namespaces \\.\ and \\?\ . The first makes an offline tool write
	// over the network; the second bypasses Win32 path normalisation, which
	// would defeat any containment check that relies on it.
	if strings.HasPrefix(normalised, "//") {
		return newFieldError(ErrInvalidValue, field, dir,
			"must not be a UNC or device path")
	}
	// "C:foo" is drive-relative: it resolves against a per-drive working
	// directory, so the target depends on process state the user cannot see.
	if len(normalised) >= 2 && normalised[1] == ':' && isASCIILetter(normalised[0]) &&
		len(normalised) > 2 && normalised[2] != '/' {
		return newFieldError(ErrInvalidValue, field, dir,
			"must not be drive-relative; give a full path such as C:/projects")
	}

	cleaned := path.Clean(normalised)
	if isFilesystemRoot(cleaned) {
		return newFieldError(ErrInvalidValue, field, dir,
			"must not be a filesystem root")
	}

	// Every segment must be usable as a directory name. A reserved device
	// name silently discards writes rather than failing, so a repository
	// would be reported as created without existing.
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "" || segment == "." || strings.HasSuffix(segment, ":") {
			continue
		}
		if isWindowsReserved(segment) {
			return newFieldError(ErrUnsafeName, field, dir,
				"contains "+segment+", a reserved device name on Windows")
		}
		if strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return newFieldError(ErrUnsafeName, field, dir,
				"has a path segment ending in '.' or a space")
		}
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

// ValidateNoControlCharacters reports whether s is free of control characters.
//
// A control character in a value that is later printed lets that value forge
// additional lines of output: a newline in a description can fabricate rows in
// the confirmation summary, and an escape sequence can rewrite what the user
// sees before they agree to it. The confirmation block is the gate that
// protects an existing directory, so nothing that can rewrite it may enter a
// ProjectConfig.
func ValidateNoControlCharacters(field, s string) error {
	for _, r := range s {
		if unicode.IsControl(r) {
			// The raw value is passed through: FieldError renders it with %q,
			// which escapes the control characters rather than replaying them
			// into the error output.
			return newFieldError(ErrInvalidValue, field, s,
				"must not contain control characters")
		}
	}
	return nil
}

// remoteSchemes are the transports a generated repository may use as origin.
// The list is an allowlist because Git's remote syntax includes transports
// that execute a command — ext:: runs a shell command, fd:: reads a file
// descriptor — and a remote URL is data the tool will later hand to git.
var remoteSchemes = []string{"https://", "http://", "ssh://", "git://", "file://"}

// ValidateRemote reports whether remote is safe to register as origin.
//
// An empty remote is valid and means "do not add one". The rules reject a
// value that git would read as an option rather than a URL, a transport that
// can execute a command, and anything that is neither a recognised scheme,
// an scp-style host:path, nor a local path.
func ValidateRemote(remote string) error {
	const field = "Remote"

	trimmed := strings.TrimSpace(remote)
	if trimmed == "" {
		return nil
	}
	if err := ValidateNoControlCharacters(field, trimmed); err != nil {
		return err
	}
	// A leading dash makes git read the value as a flag, whatever follows it.
	if strings.HasPrefix(trimmed, "-") {
		return newFieldError(ErrInvalidValue, field, remote,
			"must not begin with '-'")
	}
	lower := strings.ToLower(trimmed)
	for _, scheme := range []string{"ext::", "fd::"} {
		if strings.HasPrefix(lower, scheme) {
			return newFieldError(ErrInvalidValue, field, remote,
				"uses the "+strings.TrimSuffix(scheme, "::")+" transport, which executes a command")
		}
	}

	for _, scheme := range remoteSchemes {
		if strings.HasPrefix(lower, scheme) {
			return nil
		}
	}
	// scp-style: user@host:path, which git accepts and which has no scheme.
	if i := strings.Index(trimmed, ":"); i > 0 && strings.Contains(trimmed[:i], "@") {
		return nil
	}
	// A local path is legitimate for a remote that is a directory.
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, ".") || isWindowsAbsolute(trimmed) {
		return nil
	}
	return newFieldError(ErrInvalidValue, field, remote,
		"must be an https, ssh, git or file URL, a user@host:path address, or a local path")
}

// isWindowsAbsolute reports whether p starts with a drive letter and a
// separator, such as C:\repos.
func isWindowsAbsolute(p string) bool {
	return len(p) >= 3 && p[1] == ':' && isASCIILetter(p[0]) && (p[2] == '\\' || p[2] == '/')
}
