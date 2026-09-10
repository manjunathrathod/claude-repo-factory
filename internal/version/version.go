// Package version exposes the build identity of the binary.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// These values are overridden at build time with -ldflags -X.
var (
	// Version is the semantic version of the release.
	Version = "0.1.0-dev"
	// Commit is the git commit the binary was built from.
	Commit = ""
	// Date is the build timestamp in RFC 3339 form.
	Date = ""
)

// Info is the resolved build identity.
type Info struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
	Platform  string
}

// Get returns the build identity, falling back to the values Go embeds in
// the binary when the ldflags were not supplied.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	if info.Commit == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range bi.Settings {
				switch setting.Key {
				case "vcs.revision":
					info.Commit = setting.Value
				case "vcs.time":
					if info.Date == "" {
						info.Date = setting.Value
					}
				}
			}
		}
	}
	return info
}

// String renders a single line summary, as printed by the version command.
func (i Info) String() string {
	s := fmt.Sprintf("claude-repo-factory %s", i.Version)
	if i.Commit != "" {
		short := i.Commit
		if len(short) > 12 {
			short = short[:12]
		}
		s += " (" + short + ")"
	}
	return s + fmt.Sprintf(" %s %s", i.GoVersion, i.Platform)
}
