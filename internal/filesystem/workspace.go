// Package filesystem creates directories on behalf of the factory, and is the
// only place in the tool that does.
//
// Everything here is written against one rule: a repository must be created
// inside the directory the user chose and nowhere else. The package therefore
// knows nothing about repositories, languages or configuration — it takes a
// base directory and a single directory name, and refuses anything that could
// resolve outside the base.
//
// Confinement is delegated to os.Root rather than reimplemented with string
// comparison. A prefix check on a cleaned path is defeated by a symlink inside
// the base, and on Windows by the device namespace; os.Root holds a handle to
// the base directory and rejects any name that escapes it, symlinks included.
package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// DefaultDirMode is the permission new directories are created with. It is
// deliberately not world-writable.
const DefaultDirMode fs.FileMode = 0o755

// MaxSegmentLength bounds a directory name, well under the 255 bytes a
// filesystem component usually allows, because the name also becomes part of
// longer paths beneath it.
const MaxSegmentLength = 64

// Failures are grouped under these sentinels so a caller can react to a
// category without matching on message text.
var (
	// ErrNotEmpty is returned when the target directory already exists and
	// contains something. The factory never overwrites existing work.
	ErrNotEmpty = errors.New("target directory is not empty")
	// ErrUnsafeName is returned when the directory name is not a single,
	// ordinary path segment.
	ErrUnsafeName = errors.New("unsafe directory name")
	// ErrNotDirectory is returned when the target exists but is a file.
	ErrNotDirectory = errors.New("target exists and is not a directory")
	// ErrBaseRedirected is returned when the output directory resolves,
	// through a symlink, to somewhere other than where it was asked for.
	ErrBaseRedirected = errors.New("output directory resolves elsewhere")
)

// Result describes what Create did, so a caller can report it accurately
// rather than guessing.
type Result struct {
	// Path is the absolute path of the repository directory.
	Path string
	// Created reports whether the directory was made. It is false when an
	// existing empty directory was adopted instead.
	Created bool
}

// Workspace creates repository directories.
//
// The zero Workspace is usable and creates directories with DefaultDirMode.
type Workspace struct {
	// DirMode overrides the permission new directories are created with.
	// Zero means DefaultDirMode.
	DirMode fs.FileMode
}

// dirMode returns the configured permission, defaulting when unset.
func (w Workspace) dirMode() fs.FileMode {
	if w.DirMode == 0 {
		return DefaultDirMode
	}
	return w.DirMode
}

// Create makes the directory called name inside base and returns its absolute
// path.
//
// base is created if it does not exist; name must be a single path segment, so
// a repository can never be placed anywhere but directly inside base. An
// existing empty directory is adopted rather than treated as an error, which
// is what makes re-running the command after a cancelled attempt work. An
// existing non-empty directory is refused with ErrNotEmpty: overwriting
// someone's work is never the right default.
func (w Workspace) Create(ctx context.Context, base, name string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := ValidateSegment(name); err != nil {
		return Result{}, err
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return Result{}, fmt.Errorf("resolve base directory %s: %w", base, err)
	}
	// The base is the boundary, so it has to exist before it can be opened as
	// one. Creating it here is what lets --dir name a directory that is not
	// there yet.
	if mkErr := os.MkdirAll(absBase, w.dirMode()); mkErr != nil {
		return Result{}, fmt.Errorf("create base directory %s: %w", absBase, mkErr)
	}

	root, err := os.OpenRoot(absBase)
	if err != nil {
		return Result{}, fmt.Errorf("open base directory %s: %w", absBase, err)
	}
	defer func() {
		// The handle is the confinement; failing to release it is a leak, not
		// a reason to fail a directory that was created successfully.
		_ = root.Close()
	}()

	// os.Root confines a name relative to the root; it says nothing about how
	// the root itself was chosen. MkdirAll above follows symlinks, so on a
	// shared machine another user could swap a component of absBase between
	// that call and OpenRoot and re-anchor the root somewhere else. Comparing
	// the resolved path with the requested one closes that window.
	if err := verifyBaseNotRedirected(absBase); err != nil {
		return Result{}, err
	}

	target := filepath.Join(absBase, name)

	switch info, statErr := root.Stat(name); {
	case statErr == nil && !info.IsDir():
		return Result{}, fmt.Errorf("%s: %w", target, ErrNotDirectory)

	case statErr == nil:
		empty, emptyErr := isEmpty(root, name)
		if emptyErr != nil {
			return Result{}, fmt.Errorf("inspect %s: %w", target, emptyErr)
		}
		if !empty {
			return Result{}, fmt.Errorf("%s: %w", target, ErrNotEmpty)
		}
		// Adopting an empty directory writes nothing, so there is nothing to
		// undo if a later step fails.
		return Result{Path: target, Created: false}, nil

	case !errors.Is(statErr, fs.ErrNotExist):
		return Result{}, fmt.Errorf("inspect %s: %w", target, statErr)
	}

	if err := root.Mkdir(name, w.dirMode()); err != nil {
		return Result{}, fmt.Errorf("create %s: %w", target, err)
	}
	return Result{Path: target, Created: true}, nil
}

// verifyBaseNotRedirected reports an error when the base directory resolves,
// through symlinks, to somewhere other than where it was asked for.
//
// A repository is meant to land where the user said. If a component of the
// path is a link, the directory is physically created elsewhere while the
// reported path claims otherwise — which is both a surprise and, on a machine
// with other users, a way to redirect the write. Saying so is better than
// resolving silently, because the user can then decide whether the link was
// intended.
func verifyBaseNotRedirected(absBase string) error {
	resolved, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		// A base that cannot be resolved is not usable as a boundary.
		return fmt.Errorf("resolve base directory %s: %w", absBase, err)
	}
	// A case-insensitive filesystem can differ only in spelling; that is not
	// a redirection.
	if resolved == absBase || strings.EqualFold(resolved, absBase) {
		return nil
	}
	return fmt.Errorf("%w: %s resolves to %s", ErrBaseRedirected, absBase, resolved)
}

// isEmpty reports whether the directory called name inside root has no
// entries. It reads a single entry rather than the whole listing, because the
// answer is known as soon as one is found.
func isEmpty(root *os.Root, name string) (bool, error) {
	dir, err := root.Open(name)
	if err != nil {
		return false, err
	}
	defer func() { _ = dir.Close() }()

	entries, err := dir.ReadDir(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// reservedDeviceNames cannot be used as a file or directory name on Windows,
// with or without an extension. They are rejected on every platform so a
// repository created on one stays usable on another.
var reservedDeviceNames = map[string]struct{}{
	"con": {}, "prn": {}, "aux": {}, "nul": {},
	"com1": {}, "com2": {}, "com3": {}, "com4": {}, "com5": {},
	"com6": {}, "com7": {}, "com8": {}, "com9": {},
	"lpt1": {}, "lpt2": {}, "lpt3": {}, "lpt4": {}, "lpt5": {},
	"lpt6": {}, "lpt7": {}, "lpt8": {}, "lpt9": {},
}

// isReservedDeviceName reports whether name is a Windows device name,
// ignoring case and any extension: both "NUL" and "nul.txt" are reserved.
func isReservedDeviceName(name string) bool {
	base := name
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	_, reserved := reservedDeviceNames[strings.ToLower(base)]
	return reserved
}

// ValidateSegment reports whether name is a single, ordinary directory name.
//
// It is exported because it is the rule that makes the rest safe, and a caller
// may want to apply it before doing work that would be wasted. Anything that
// could address a different directory is refused. os.Root would refuse an
// escape anyway; rejecting it here means the caller gets an explanation
// instead of a syscall error, and it keeps this package's definition of "safe"
// at least as strict as config.ValidateProjectName — otherwise the two-layer
// defence would only hold in one direction.
func ValidateSegment(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("%w: must not be empty", ErrUnsafeName)
	case name != strings.TrimSpace(name):
		return fmt.Errorf("%w: %q must not begin or end with whitespace", ErrUnsafeName, name)
	case name == "." || name == "..":
		return fmt.Errorf("%w: %q addresses a directory other than itself", ErrUnsafeName, name)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("%w: %q must be a single directory name, not a path", ErrUnsafeName, name)
	case strings.Contains(name, ".."):
		return fmt.Errorf("%w: %q must not contain %q", ErrUnsafeName, name, "..")
	case filepath.IsAbs(name) || filepath.VolumeName(name) != "":
		return fmt.Errorf("%w: %q must be relative to the output directory", ErrUnsafeName, name)
	case strings.ContainsFunc(name, unicode.IsControl):
		// Rejecting these is what lets the errors above print paths plainly: a
		// name that cannot hold a newline or an escape sequence cannot forge a
		// line of output when it is echoed back.
		return fmt.Errorf("%w: %q must not contain control characters", ErrUnsafeName, name)
	case strings.Contains(name, ":"):
		// filepath.VolumeName only sees a colon at index 1, so "widget:evil"
		// would otherwise pass and name an NTFS alternate data stream.
		return fmt.Errorf("%w: %q must not contain %q", ErrUnsafeName, name, ":")
	case strings.HasSuffix(name, "."):
		// Windows silently strips a trailing dot, so the directory created
		// would not be the one the caller was told about.
		return fmt.Errorf("%w: %q must not end with %q", ErrUnsafeName, name, ".")
	case isReservedDeviceName(name):
		return fmt.Errorf("%w: %q is a reserved device name on Windows", ErrUnsafeName, name)
	case len(name) > MaxSegmentLength:
		return fmt.Errorf("%w: %q must be %d characters or fewer", ErrUnsafeName, name, MaxSegmentLength)
	}
	return nil
}
