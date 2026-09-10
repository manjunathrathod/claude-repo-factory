package filesystem_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/filesystem"
)

func TestCreateMakesTheDirectory(t *testing.T) {
	base := t.TempDir()
	var w filesystem.Workspace

	got, err := w.Create(context.Background(), base, "payment-api")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	want := filepath.Join(base, "payment-api")
	if got.Path != want {
		t.Errorf("Path = %q, want %q", got.Path, want)
	}
	if !got.Created {
		t.Error("Created = false, want true for a directory that did not exist")
	}
	if info, statErr := os.Stat(want); statErr != nil {
		t.Fatalf("stat created directory: %v", statErr)
	} else if !info.IsDir() {
		t.Error("created path is not a directory")
	}
}

func TestCreateMakesAMissingBaseDirectory(t *testing.T) {
	// --dir may name a directory that does not exist yet; the base is created
	// so the user does not have to prepare it by hand.
	base := filepath.Join(t.TempDir(), "Projects", "team")
	var w filesystem.Workspace

	got, err := w.Create(context.Background(), base, "widget")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, statErr := os.Stat(got.Path); statErr != nil {
		t.Fatalf("stat created directory: %v", statErr)
	}
}

func TestCreateAdoptsAnExistingEmptyDirectory(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "widget")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	var w filesystem.Workspace
	got, err := w.Create(context.Background(), base, "widget")
	if err != nil {
		t.Fatalf("Create() error = %v, want an empty directory to be adopted", err)
	}
	if got.Created {
		t.Error("Created = true, want false when an existing directory was adopted")
	}
	if got.Path != target {
		t.Errorf("Path = %q, want %q", got.Path, target)
	}
}

func TestCreateRefusesANonEmptyDirectory(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, target string)
	}{
		{
			name: "containing a file",
			prepare: func(t *testing.T, target string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("hi"), 0o600); err != nil {
					t.Fatalf("prepare: %v", err)
				}
			},
		},
		{
			name: "containing a subdirectory",
			prepare: func(t *testing.T, target string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(target, "src"), 0o755); err != nil {
					t.Fatalf("prepare: %v", err)
				}
			},
		},
		{
			name: "containing a dotfile",
			prepare: func(t *testing.T, target string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(target, ".git"), []byte("x"), 0o600); err != nil {
					t.Fatalf("prepare: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			target := filepath.Join(base, "widget")
			if err := os.Mkdir(target, 0o755); err != nil {
				t.Fatalf("prepare: %v", err)
			}
			tt.prepare(t, target)

			var w filesystem.Workspace
			_, err := w.Create(context.Background(), base, "widget")
			if !errors.Is(err, filesystem.ErrNotEmpty) {
				t.Fatalf("Create() error = %v, want ErrNotEmpty", err)
			}
		})
	}
}

func TestCreateDoesNotDisturbANonEmptyDirectory(t *testing.T) {
	// Refusing is not enough: the existing content must be exactly as it was.
	base := t.TempDir()
	target := filepath.Join(base, "widget")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	existing := filepath.Join(target, "README.md")
	const content = "someone else's work"
	if err := os.WriteFile(existing, []byte(content), 0o600); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	var w filesystem.Workspace
	if _, err := w.Create(context.Background(), base, "widget"); err == nil {
		t.Fatal("Create() = nil error, want a refusal")
	}

	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(got) != content {
		t.Errorf("existing file = %q, want it untouched (%q)", got, content)
	}
}

func TestCreateRefusesAFileAtTheTarget(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "widget"), []byte("x"), 0o600); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	var w filesystem.Workspace
	_, err := w.Create(context.Background(), base, "widget")
	if !errors.Is(err, filesystem.ErrNotDirectory) {
		t.Fatalf("Create() error = %v, want ErrNotDirectory", err)
	}
}

func TestCreateRejectsUnsafeNames(t *testing.T) {
	// A name is a single directory inside the base. Anything that could
	// address somewhere else is refused before a syscall is attempted.
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "whitespace only", input: "   "},
		{name: "leading whitespace", input: " widget"},
		{name: "current directory", input: "."},
		{name: "parent directory", input: ".."},
		{name: "traversal", input: "../escaped"},
		{name: "windows traversal", input: `..\escaped`},
		{name: "nested path", input: "team/widget"},
		{name: "nested windows path", input: `team\widget`},
		{name: "embedded traversal", input: "team/../../escaped"},
		{name: "posix absolute", input: "/etc"},
		{name: "windows absolute", input: `C:\Windows`},
		{name: "drive relative", input: "C:widget"},
		{name: "UNC", input: `\\server\share`},
		{name: "hidden traversal", input: "a..b/../.."},
		{name: "newline", input: "wid\nget"},
		{name: "carriage return", input: "wid\rget"},
		{name: "ANSI escape", input: "widget\x1b[2J"},
		{name: "NUL", input: "wid\x00get"},
		{name: "alternate data stream", input: "widget:evil"},
		{name: "trailing dot", input: "widget."},
		{name: "reserved device name", input: "nul"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			var w filesystem.Workspace

			_, err := w.Create(context.Background(), base, tt.input)
			if !errors.Is(err, filesystem.ErrUnsafeName) {
				t.Fatalf("Create(%q) error = %v, want ErrUnsafeName", tt.input, err)
			}
		})
	}
}

func TestCreateTouchesNothingButTheTarget(t *testing.T) {
	// The strongest statement this package can make: for a successful run the
	// only change anywhere under the base is the one directory that was asked
	// for, and for a refused run there is no change at all.
	tests := []struct {
		name    string
		project string
		wantErr bool
		wantNew []string
	}{
		{name: "success adds exactly one directory", project: "widget", wantNew: []string{"widget"}},
		{name: "traversal changes nothing", project: "../escaped", wantErr: true},
		{name: "nested path changes nothing", project: "a/b", wantErr: true},
		{name: "absolute path changes nothing", project: `C:\Windows`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			base := filepath.Join(root, "base")
			sibling := filepath.Join(root, "sibling")
			for _, d := range []string{base, sibling} {
				if err := os.Mkdir(d, 0o755); err != nil {
					t.Fatalf("prepare: %v", err)
				}
			}
			if err := os.WriteFile(filepath.Join(sibling, "keep.txt"), []byte("keep"), 0o600); err != nil {
				t.Fatalf("prepare: %v", err)
			}
			before := treeOf(t, root)

			var w filesystem.Workspace
			_, err := w.Create(context.Background(), base, tt.project)
			if tt.wantErr && err == nil {
				t.Fatalf("Create(%q) = nil error, want a refusal", tt.project)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Create(%q) error = %v", tt.project, err)
			}

			after := treeOf(t, root)
			// Compared both ways: additions alone would let a Create that
			// deleted a sibling pass, which the name of this test forbids.
			if gone := added(after, before); len(gone) != 0 {
				t.Errorf("Create removed %v", gone)
			}
			if got, readErr := os.ReadFile(filepath.Join(sibling, "keep.txt")); readErr != nil {
				t.Errorf("sibling file unreadable after Create: %v", readErr)
			} else if string(got) != "keep" {
				t.Errorf("sibling file = %q, want it untouched", got)
			}

			gotNew := added(before, after)
			want := make([]string, 0, len(tt.wantNew))
			for _, n := range tt.wantNew {
				want = append(want, "base/"+n)
			}
			if strings.Join(gotNew, ",") != strings.Join(want, ",") {
				t.Errorf("new paths = %v, want %v", gotNew, want)
			}
		})
	}
}

func TestCreateHonoursACancelledContext(t *testing.T) {
	base := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var w filesystem.Workspace
	if _, err := w.Create(ctx, base, "widget"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	if entries, err := os.ReadDir(base); err != nil {
		t.Fatalf("read base: %v", err)
	} else if len(entries) != 0 {
		t.Errorf("a cancelled call created %d entries, want none", len(entries))
	}
}

func TestCreateIsRepeatable(t *testing.T) {
	// Re-running after a cancelled attempt must succeed rather than trip over
	// the empty directory the first attempt left behind.
	base := t.TempDir()
	var w filesystem.Workspace

	first, err := w.Create(context.Background(), base, "widget")
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := w.Create(context.Background(), base, "widget")
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if first.Path != second.Path {
		t.Errorf("paths differ: %q then %q", first.Path, second.Path)
	}
	if second.Created {
		t.Error("second Created = true, want false")
	}
}

func TestCreateUsesANonWorldWritableMode(t *testing.T) {
	// Windows has no Unix permission bits: Go reports 0777 for every directory
	// there, so the assertion is only meaningful on Unix.
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not modelled on Windows")
	}

	base := t.TempDir()
	var w filesystem.Workspace

	got, err := w.Create(context.Background(), base, "widget")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	info, err := os.Stat(got.Path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o002 != 0 {
		t.Errorf("mode = %v, want no world-write bit", perm)
	}
}

func TestCreateReportsABaseReachedThroughASymlink(t *testing.T) {
	// The package claims confinement; a base that is itself a link would put
	// the repository somewhere other than the path reported back.
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows needs elevation")
	}

	root := t.TempDir()
	actual := filepath.Join(root, "actual")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(actual, 0o755); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := os.Symlink(actual, link); err != nil {
		t.Fatalf("prepare symlink: %v", err)
	}

	var w filesystem.Workspace
	_, err := w.Create(context.Background(), link, "widget")
	if !errors.Is(err, filesystem.ErrBaseRedirected) {
		t.Fatalf("Create() error = %v, want ErrBaseRedirected", err)
	}
	entries, readErr := os.ReadDir(actual)
	if readErr != nil {
		t.Fatalf("read actual directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("a redirected base still created %d entries", len(entries))
	}
}

func TestCreateRefusesToFollowASymlinkedTarget(t *testing.T) {
	// A link at base/name pointing outside base must not become a way to write
	// there. os.Root enforces this; this proves we rely on it correctly rather
	// than falling through to Mkdir.
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows needs elevation")
	}

	root := t.TempDir()
	base := filepath.Join(root, "base")
	outside := filepath.Join(root, "outside")
	for _, d := range []string{base, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatalf("prepare: %v", err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(base, "widget")); err != nil {
		t.Fatalf("prepare symlink: %v", err)
	}

	var w filesystem.Workspace
	if _, err := w.Create(context.Background(), base, "widget"); err == nil {
		t.Fatal("Create() = nil error, want the escaping symlink to be refused")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatalf("read outside directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("wrote %d entries through a symlink that escapes the base", len(entries))
	}
}

func TestValidateSegment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "simple", input: "widget"},
		{name: "dashes", input: "payment-api"},
		{name: "underscores", input: "my_widget"},
		{name: "dots", input: "acme.widget"},
		{name: "single dot inside", input: "v1.2"},
		{name: "digits", input: "2fa"},
		{name: "com0 is not reserved", input: "com0"},
		{name: "at the length limit", input: strings.Repeat("a", filesystem.MaxSegmentLength)},

		{name: "empty", input: "", wantErr: true},
		{name: "dot", input: ".", wantErr: true},
		{name: "dotdot", input: "..", wantErr: true},
		{name: "slash", input: "a/b", wantErr: true},
		{name: "backslash", input: `a\b`, wantErr: true},
		{name: "traversal", input: "../x", wantErr: true},
		{name: "embedded dotdot", input: "a..b", wantErr: true},
		{name: "absolute", input: "/x", wantErr: true},
		{name: "volume", input: "C:x", wantErr: true},
		{name: "newline", input: "a\nb", wantErr: true},
		{name: "escape sequence", input: "a\x1bb", wantErr: true},
		{name: "alternate data stream", input: "widget:evil", wantErr: true},
		{name: "trailing colon", input: "widget:", wantErr: true},
		{name: "trailing dot", input: "widget.", wantErr: true},
		{name: "reserved device name", input: "nul", wantErr: true},
		{name: "reserved uppercase", input: "CON", wantErr: true},
		{name: "reserved with extension", input: "nul.txt", wantErr: true},
		{name: "reserved lpt9", input: "lpt9", wantErr: true},
		{name: "over the length limit", input: strings.Repeat("a", filesystem.MaxSegmentLength+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := filesystem.ValidateSegment(tt.input)
			if tt.wantErr && !errors.Is(err, filesystem.ErrUnsafeName) {
				t.Fatalf("ValidateSegment(%q) = %v, want ErrUnsafeName", tt.input, err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateSegment(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

// treeOf returns every path under root, relative and slash-separated, sorted.
func treeOf(t *testing.T, root string) []string {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel != "." {
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(paths)
	return paths
}

// added returns the paths present in after but not before.
func added(before, after []string) []string {
	seen := make(map[string]struct{}, len(before))
	for _, p := range before {
		seen[p] = struct{}{}
	}
	var out []string
	for _, p := range after {
		if _, ok := seen[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
