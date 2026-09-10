package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/cli"
	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/generator"
	"github.com/manjunathrathod/claude-repo-factory/internal/lang"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/prompt"
)

// fakePath is what the stub reports as the created directory. It is a fixed
// sentinel rather than ResolvedProjectDirectory: computing it the way the
// production code does would make any test asserting on the printed path agree
// with whatever that method happens to do, bug included.
const fakePath = "<fake>/created-directory"

// fakeGenerator records what the command asked to be created without touching
// a filesystem. It is the default for CLI tests: the command layer's job is to
// resolve a configuration and hand it over, and a test of that should not be
// able to litter the working tree.
type fakeGenerator struct {
	// configs records every configuration Prepare was given, in order.
	configs []config.ProjectConfig
	// err, when set, is returned instead of creating anything.
	err error
	// created and gitInitialized override the reported flags.
	created        bool
	gitInitialized bool
}

func newFakeGenerator() *fakeGenerator { return &fakeGenerator{created: true, gitInitialized: true} }

func (f *fakeGenerator) Prepare(_ context.Context, cfg config.ProjectConfig, _ config.Catalog) (generator.Result, error) {
	f.configs = append(f.configs, cfg)
	if f.err != nil {
		return generator.Result{}, f.err
	}
	return generator.Result{
		Path:           fakePath,
		Created:        f.created,
		GitInitialized: f.gitInitialized && cfg.InitializeGit,
	}, nil
}

// run executes the command tree with the given arguments and returns what it
// wrote to stdout and stderr. Creation is faked; use runWith to inspect or
// replace the generator.
func run(t *testing.T, asker prompt.Asker, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runWith(t, asker, newFakeGenerator(), args...)
}

// runWith is run with an explicit workspace preparer, for tests that assert on
// what the command asked for or that drive the real filesystem.
func runWith(t *testing.T, asker prompt.Asker, gen cli.Preparer, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var out, errOut bytes.Buffer
	app := &cli.App{
		Registry:  lang.Registry(),
		Asker:     asker,
		Out:       &out,
		Err:       &errOut,
		Generator: gen,
	}
	root := cli.NewRootCommand(app)
	root.SetArgs(args)
	root.SetOut(&out)
	root.SetErr(&errOut)

	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestLanguagesListsStableAndPlanned(t *testing.T) {
	out, _, err := run(t, nil, "languages")
	if err != nil {
		t.Fatalf("languages error = %v", err)
	}

	for _, want := range []string{"nodejs", "python", "go", "java", "rust", "terraform", "nextjs", "stable", "planned"} {
		if !strings.Contains(out, want) {
			t.Errorf("languages output is missing %q\n%s", want, out)
		}
	}
}

func TestLanguagesJSON(t *testing.T) {
	out, _, err := run(t, nil, "languages", "--json")
	if err != nil {
		t.Fatalf("languages --json error = %v", err)
	}

	var descriptors []plugin.Descriptor
	if err := json.Unmarshal([]byte(out), &descriptors); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if len(descriptors) < 4 {
		t.Fatalf("got %d descriptors, want at least the four supported languages", len(descriptors))
	}
}

func TestLanguagesWithoutPlanned(t *testing.T) {
	out, _, err := run(t, nil, "languages", "--all=false")
	if err != nil {
		t.Fatalf("languages --all=false error = %v", err)
	}

	if strings.Contains(out, "planned") {
		t.Errorf("output still lists planned languages\n%s", out)
	}
	if !strings.Contains(out, "python") {
		t.Errorf("output is missing the supported languages\n%s", out)
	}
}

func TestVersion(t *testing.T) {
	out, _, err := run(t, nil, "version")
	if err != nil {
		t.Fatalf("version error = %v", err)
	}
	if !strings.Contains(out, "claude-repo-factory") {
		t.Errorf("version output = %q, want the binary name", out)
	}
}

func TestVersionJSON(t *testing.T) {
	out, _, err := run(t, nil, "version", "--json")
	if err != nil {
		t.Fatalf("version --json error = %v", err)
	}

	var info map[string]any
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	for _, key := range []string{"Version", "GoVersion", "Platform"} {
		if _, ok := info[key]; !ok {
			t.Errorf("version JSON is missing %q", key)
		}
	}
}
