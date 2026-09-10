package version_test

import (
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/version"
)

func TestGetAlwaysReportsTheRuntime(t *testing.T) {
	info := version.Get()

	if info.Version == "" {
		t.Error("Version is empty")
	}
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Errorf("GoVersion = %q, want a go... string", info.GoVersion)
	}
	if !strings.Contains(info.Platform, "/") {
		t.Errorf("Platform = %q, want os/arch", info.Platform)
	}
}

func TestStringShortensTheCommit(t *testing.T) {
	info := version.Info{
		Version:   "1.2.3",
		Commit:    "0123456789abcdef0123456789abcdef01234567",
		GoVersion: "go1.24.3",
		Platform:  "linux/amd64",
	}

	got := info.String()
	for _, want := range []string{"claude-repo-factory 1.2.3", "0123456789ab", "go1.24.3", "linux/amd64"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
	if strings.Contains(got, "0123456789abc") {
		t.Errorf("String() = %q, want the commit truncated to 12 characters", got)
	}
}

func TestStringWithoutACommit(t *testing.T) {
	info := version.Info{Version: "1.2.3", GoVersion: "go1.24.3", Platform: "linux/amd64"}

	if got := info.String(); strings.Contains(got, "()") {
		t.Errorf("String() = %q, want no empty commit parentheses", got)
	}
}
