package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/prompt"
)

// The prompt messages the create command uses. They are duplicated here on
// purpose: a test that imported the constants would still pass if the wording
// changed, and the wording is part of the user-facing contract.
const (
	askName        = "Project name"
	askDescription = "Description"
	askLanguage    = "Programming language"
	askProjectType = "Project type"
	askPackageMgr  = "Package manager"
	askOutputDir   = "Output directory"
	askGit         = "Initialize Git?"
	askClaude      = "Include Claude Code setup?"
	askActions     = "Include GitHub Actions?"
	askConfirm     = "Create this project?"
	askGoModule    = "Go module path"

	// summaryTitleText heads the confirmation block. A rejected configuration
	// must never reach it.
	summaryTitleText = "Project Configuration"
)

// fullyScripted returns an asker with an answer for every question the create
// command asks for a Go project, so a test can override just the one it cares
// about.
func fullyScripted() *prompt.Scripted {
	asker := prompt.NewScripted(map[string]string{
		askName:        "widget",
		askDescription: "Widget control plane",
		askLanguage:    "go",
		askProjectType: "cli",
		askOutputDir:   "widget",
		askGoModule:    "github.com/acme/widget",
	})
	asker.Confirms = map[string]bool{
		askGit:     true,
		askClaude:  true,
		askActions: true,
		askConfirm: true,
	}
	return asker
}

func TestCreateCommandIsRegisteredUnderBothNames(t *testing.T) {
	// "create" is the documented name and "new" the compatibility alias.
	// Both must reach the same command.
	for _, name := range []string{"create", "new"} {
		t.Run(name, func(t *testing.T) {
			out, _, err := run(t, nil, name, "widget", "-l", "go", "--yes")
			if err != nil {
				t.Fatalf("%s error = %v\n%s", name, err, out)
			}
			if !strings.Contains(out, "Configuration accepted.") {
				t.Errorf("%s did not accept the configuration\n%s", name, out)
			}
		})
	}
}

func TestCreateHelpDescribesTheCommand(t *testing.T) {
	out, _, err := run(t, nil, "create", "--help")
	if err != nil {
		t.Fatalf("create --help error = %v", err)
	}
	for _, want := range []string{"--language", "--type", "--package-manager", "--dir", "--yes", "--no-git", "--no-claude", "--no-ci"} {
		if !strings.Contains(out, want) {
			t.Errorf("help is missing %q\n%s", want, out)
		}
	}
}

func TestCreatePrintsTheSummaryInTheDocumentedFormat(t *testing.T) {
	out, _, err := run(t, nil,
		"create", "payment-api",
		"--language", "node",
		"--type", "api",
		"--package-manager", "npm",
		"--description", "Payment service",
		"--dir", "services",
		"--yes",
	)
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	for _, want := range []string{
		"Project Configuration",
		"---------------------",
		"Name: payment-api",
		"Description: Payment service",
		"Language: Node.js / TypeScript",
		"Type: API",
		"Package Manager: npm",
		"Output Directory: ",
		"Initialize Git: Yes",
		"Claude Code Setup: Yes",
		"GitHub Actions: Yes",
		"Configuration accepted.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary is missing %q\n%s", want, out)
		}
	}
}

func TestCreateSummaryOrderIsStable(t *testing.T) {
	out, _, err := run(t, nil, "create", "widget", "-l", "go", "--description", "Widget", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	labels := []string{
		"Name:", "Description:", "Language:", "Type:", "Package Manager:",
		"Output Directory:", "Initialize Git:", "Claude Code Setup:", "GitHub Actions:",
	}
	last := -1
	for _, label := range labels {
		at := strings.Index(out, label)
		if at < 0 {
			t.Fatalf("summary is missing %q\n%s", label, out)
		}
		if at < last {
			t.Errorf("%q appears out of order\n%s", label, out)
		}
		last = at
	}
}

func TestCreateAsksTheDocumentedQuestionsInOrder(t *testing.T) {
	asker := fullyScripted()

	out, _, err := run(t, asker, "create")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	// Go declares a single package manager, so that question is correctly
	// skipped; the rest are asked in the documented order.
	want := []string{askName, askDescription, askLanguage, askProjectType, askOutputDir, askGit, askClaude, askActions}
	var got []string
	for _, asked := range asker.Asked {
		for _, w := range want {
			if asked == w {
				got = append(got, asked)
			}
		}
	}

	if len(got) != len(want) {
		t.Fatalf("asked %v, want all of %v", asker.Asked, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("question %d = %q, want %q (full order: %v)", i, got[i], want[i], asker.Asked)
		}
	}
}

func TestCreateAsksToConfirmLast(t *testing.T) {
	asker := fullyScripted()

	if _, _, err := run(t, asker, "create"); err != nil {
		t.Fatalf("create error = %v", err)
	}
	if len(asker.Asked) == 0 {
		t.Fatal("no questions were asked")
	}
	if last := asker.Asked[len(asker.Asked)-1]; last != askConfirm {
		t.Errorf("last question = %q, want %q", last, askConfirm)
	}
}

func TestCreateDeclinedLeavesNothingBehind(t *testing.T) {
	asker := fullyScripted()
	asker.Confirms[askConfirm] = false

	out, _, err := run(t, asker, "create")
	if err != nil {
		t.Fatalf("declining is not an error, got %v", err)
	}
	if strings.Contains(out, "Configuration accepted.") {
		t.Errorf("a declined configuration was accepted anyway\n%s", out)
	}
	if !strings.Contains(out, "Cancelled") {
		t.Errorf("declining did not report a cancellation\n%s", out)
	}
}

func TestCreateFailsWhenAPromptCannotBeAnswered(t *testing.T) {
	_, _, err := run(t, prompt.NewScripted(nil), "create")
	if err == nil {
		t.Fatal("create = nil error, want the unanswerable prompt to surface")
	}
}

// interruptingAsker aborts at the named prompt, standing in for Ctrl+C.
type interruptingAsker struct {
	*prompt.Scripted
	at string
}

func (a *interruptingAsker) Input(message, help, def string) (string, error) {
	if message == a.at {
		return "", prompt.ErrInterrupted
	}
	return a.Scripted.Input(message, help, def)
}

func (a *interruptingAsker) Confirm(message, help string, def bool) (bool, error) {
	if message == a.at {
		return false, prompt.ErrInterrupted
	}
	return a.Scripted.Confirm(message, help, def)
}

// Ctrl+C must stay matchable as prompt.ErrInterrupted all the way out of the
// command, because that is what root.Execute maps to exit code 130.
func TestCreateCancellationStaysMatchable(t *testing.T) {
	for _, at := range []string{askName, askOutputDir, askConfirm} {
		t.Run(at, func(t *testing.T) {
			asker := &interruptingAsker{Scripted: fullyScripted(), at: at}

			out, _, err := run(t, asker, "create")
			if !errors.Is(err, prompt.ErrInterrupted) {
				t.Fatalf("error = %v, want it to wrap prompt.ErrInterrupted", err)
			}
			if strings.Contains(out, "Configuration accepted.") {
				t.Errorf("a cancelled run accepted the configuration\n%s", out)
			}
		})
	}
}

func TestCreateLanguageSelection(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "canonical id", input: "go", want: "Language: Go"},
		{name: "node alias", input: "node", want: "Language: Node.js / TypeScript"},
		{name: "typescript alias", input: "typescript", want: "Language: Node.js / TypeScript"},
		{name: "python", input: "python", want: "Language: Python"},
		{name: "java", input: "java", want: "Language: Java"},
		{name: "uppercase is normalised", input: "GO", want: "Language: Go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(t, nil, "create", "widget", "-l", tt.input, "--yes")
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("summary is missing %q\n%s", tt.want, out)
			}
		})
	}
}

func TestCreateProjectTypeSelection(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "api", args: []string{"-l", "go", "-t", "api"}, want: "Type: API"},
		{name: "cli", args: []string{"-l", "go", "-t", "cli"}, want: "Type: CLI"},
		{name: "library", args: []string{"-l", "go", "-t", "library"}, want: "Type: Library"},
		{name: "worker", args: []string{"-l", "go", "-t", "worker"}, want: "Type: Worker"},
		{name: "uppercase is normalised", args: []string{"-l", "go", "-t", "API"}, want: "Type: API"},
		{name: "go defaults to cli", args: []string{"-l", "go"}, want: "Type: CLI"},
		{name: "python defaults to library", args: []string{"-l", "python"}, want: "Type: Library"},
		{name: "java defaults to api", args: []string{"-l", "java"}, want: "Type: API"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"create", "widget"}, tt.args...)
			out, _, err := run(t, nil, append(args, "--yes")...)
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("summary is missing %q\n%s", tt.want, out)
			}
		})
	}
}

func TestCreatePackageManagerSelection(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "explicit poetry", args: []string{"-l", "python", "--package-manager", "poetry"}, want: "Package Manager: poetry"},
		{name: "explicit pnpm", args: []string{"-l", "nodejs", "--package-manager", "pnpm"}, want: "Package Manager: pnpm"},
		{name: "explicit gradle", args: []string{"-l", "java", "--package-manager", "gradle"}, want: "Package Manager: gradle"},
		{name: "python defaults to uv", args: []string{"-l", "python"}, want: "Package Manager: uv"},
		{name: "node defaults to npm", args: []string{"-l", "nodejs"}, want: "Package Manager: npm"},
		{name: "java defaults to maven", args: []string{"-l", "java"}, want: "Package Manager: maven"},
		{name: "go has only one", args: []string{"-l", "go"}, want: "Package Manager: gomod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"create", "widget"}, tt.args...)
			out, _, err := run(t, nil, append(args, "--yes")...)
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("summary is missing %q\n%s", tt.want, out)
			}
		})
	}
}

func TestCreatePackageManagerChoicesFollowTheLanguage(t *testing.T) {
	// The offered set is ecosystem knowledge and must come from the plugin,
	// so a manager from another ecosystem is rejected rather than accepted.
	tests := []struct {
		name     string
		language string
		manager  string
	}{
		{name: "npm is not a python manager", language: "python", manager: "npm"},
		{name: "uv is not a node manager", language: "nodejs", manager: "uv"},
		{name: "maven is not a go manager", language: "go", manager: "maven"},
		{name: "poetry is not a java manager", language: "java", manager: "poetry"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := run(t, nil, "create", "widget", "-l", tt.language, "--package-manager", tt.manager, "--yes")
			if err == nil {
				t.Fatalf("create accepted %q for %s", tt.manager, tt.language)
			}
			if !strings.Contains(err.Error(), "does not support package manager") {
				t.Errorf("error = %v, want it to name the unsupported manager", err)
			}
		})
	}
}

func TestCreateDoesNotAskAboutASinglePackageManager(t *testing.T) {
	asker := fullyScripted()

	if _, _, err := run(t, asker, "create", "widget", "-l", "go"); err != nil {
		t.Fatalf("create error = %v", err)
	}
	if slices.Contains(asker.Asked, askPackageMgr) {
		t.Error("the user was asked to choose between one package manager")
	}
}

func TestCreateAsksAboutSeveralPackageManagers(t *testing.T) {
	asker := fullyScripted()
	asker.Answers[askLanguage] = "python"
	asker.Answers[askPackageMgr] = "pip"
	asker.Answers["Python package name"] = "widget"

	out, _, err := run(t, asker, "create", "widget")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	if !slices.Contains(asker.Asked, askPackageMgr) {
		t.Errorf("the user was never asked to choose a package manager (asked: %v)", asker.Asked)
	}
	if !strings.Contains(out, "Package Manager: pip") {
		t.Errorf("summary did not use the chosen package manager\n%s", out)
	}
}

func TestCreateDefaults(t *testing.T) {
	out, _, err := run(t, nil, "create", "widget", "-l", "go", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	// Every professional feature is on unless it was turned off.
	for _, want := range []string{"Initialize Git: Yes", "Claude Code Setup: Yes", "GitHub Actions: Yes"} {
		if !strings.Contains(out, want) {
			t.Errorf("default is not enabled: %q\n%s", want, out)
		}
	}
}

func TestCreateOutputDirectoryDefaultsToTheWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	out, _, err := run(t, nil, "create", "widget", "-l", "go", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	// The output directory is the PARENT, so it must be the working directory
	// itself and must NOT end in the project name — defaulting it to the
	// project name is what once nested the repository at <cwd>/widget/widget.
	line := summaryValue(t, out, "Output Directory")
	if !filepath.IsAbs(line) {
		t.Errorf("output directory = %q, want an absolute path", line)
	}
	if filepath.Base(line) == "widget" {
		t.Errorf("output directory = %q; the parent must not be the project name", line)
	}
	// The stated target, however, must end in the project name.
	if !strings.Contains(out, filepath.Join(line, "widget")) {
		t.Errorf("output does not state the target %q\n%s", filepath.Join(line, "widget"), out)
	}
}

func TestCreateOutputDirectoryAcceptsPlatformPaths(t *testing.T) {
	// Relative, POSIX-absolute and Windows-shaped paths must all be accepted;
	// resolution to the host form is the model's job, not the CLI's.
	for _, dir := range []string{"services", "services/team", `services\team`, "./services"} {
		t.Run(dir, func(t *testing.T) {
			out, _, err := run(t, nil, "create", "widget", "-l", "go", "--dir", dir, "--yes")
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			if !strings.Contains(out, "Output Directory:") {
				t.Errorf("summary is missing the output directory\n%s", out)
			}
		})
	}
}

func TestCreateFeatureFlagsOverrideTheDefaults(t *testing.T) {
	tests := []struct {
		name string
		flag string
		want string
	}{
		{name: "no git", flag: "--no-git", want: "Initialize Git: No"},
		{name: "no claude", flag: "--no-claude", want: "Claude Code Setup: No"},
		{name: "no ci", flag: "--no-ci", want: "GitHub Actions: No"},
		// An explicit =false must keep the feature on. Without newOptions.explicit
		// these three cases are indistinguishable from the flag being absent.
		{name: "explicit no-git=false keeps git on", flag: "--no-git=false", want: "Initialize Git: Yes"},
		{name: "explicit no-claude=false keeps claude on", flag: "--no-claude=false", want: "Claude Code Setup: Yes"},
		{name: "explicit no-ci=false keeps actions on", flag: "--no-ci=false", want: "GitHub Actions: Yes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(t, nil, "create", "widget", "-l", "go", "--yes", tt.flag)
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("summary is missing %q\n%s", tt.want, out)
			}
		})
	}
}

func TestCreateDoesNotAskAboutAFeatureGivenAsAFlag(t *testing.T) {
	tests := []struct {
		name   string
		flag   string
		prompt string
	}{
		{name: "git", flag: "--no-git", prompt: askGit},
		{name: "claude", flag: "--no-claude", prompt: askClaude},
		{name: "actions", flag: "--no-ci", prompt: askActions},
		{name: "git given as false", flag: "--no-git=false", prompt: askGit},
		{name: "claude given as false", flag: "--no-claude=false", prompt: askClaude},
		{name: "actions given as false", flag: "--no-ci=false", prompt: askActions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asker := fullyScripted()
			if _, _, err := run(t, asker, "create", "widget", "-l", "go", tt.flag); err != nil {
				t.Fatalf("create error = %v", err)
			}
			if slices.Contains(asker.Asked, tt.prompt) {
				t.Errorf("the user was asked %q even though %s was passed", tt.prompt, tt.flag)
			}
		})
	}
}

func TestCreateDoesNotPromptForAnswersGivenAsFlags(t *testing.T) {
	asker := fullyScripted()

	_, _, err := run(t, asker,
		"create", "widget",
		"-l", "go",
		"-t", "cli",
		"--description", "Widget control plane",
		"--dir", "services",
		"--set", "go_module=github.com/acme/widget",
	)
	if err != nil {
		t.Fatalf("create error = %v", err)
	}

	for _, unwanted := range []string{askName, askLanguage, askProjectType, askDescription, askOutputDir, askGoModule} {
		if slices.Contains(asker.Asked, unwanted) {
			t.Errorf("the user was asked %q even though it was passed as a flag", unwanted)
		}
	}
}

func TestCreateWithYesNeverPrompts(t *testing.T) {
	// A scripted asker with no answers errors on any question, so reaching
	// the end proves --yes asked nothing.
	asker := prompt.NewScripted(nil)

	out, _, err := run(t, asker, "create", "widget", "-l", "go", "--yes")
	if err != nil {
		t.Fatalf("create --yes error = %v\n%s", err, out)
	}
	if len(asker.Asked) != 0 {
		t.Errorf("--yes asked %v, want nothing", asker.Asked)
	}
	if !strings.Contains(out, "Configuration accepted.") {
		t.Errorf("--yes did not accept the configuration\n%s", out)
	}
}

func TestCreateMapsAnswersOntoTheConfiguration(t *testing.T) {
	asker := fullyScripted()
	asker.Answers[askName] = "payment-api"
	asker.Answers[askDescription] = "Payment service"
	asker.Answers[askLanguage] = "nodejs"
	asker.Answers[askProjectType] = "api"
	asker.Answers[askPackageMgr] = "pnpm"
	asker.Answers[askOutputDir] = "services"
	asker.Answers["npm package name"] = "payment-api"
	asker.Confirms[askActions] = false

	out, _, err := run(t, asker, "create")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}

	for _, want := range []string{
		"Name: payment-api",
		"Description: Payment service",
		"Language: Node.js / TypeScript",
		"Type: API",
		"Package Manager: pnpm",
		"Initialize Git: Yes",
		"Claude Code Setup: Yes",
		"GitHub Actions: No",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary is missing %q\n%s", want, out)
		}
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "unknown language",
			args:    []string{"create", "widget", "-l", "cobol", "--yes"},
			wantErr: "unknown language",
		},
		{
			name:    "planned language",
			args:    []string{"create", "widget", "-l", "rust", "--yes"},
			wantErr: "not available yet",
		},
		{
			name:    "unknown project type",
			args:    []string{"create", "widget", "-l", "go", "-t", "mainframe", "--yes"},
			wantErr: "must be one of api, cli, library, worker",
		},
		{
			name:    "project name with a separator",
			args:    []string{"create", "acme/widget", "-l", "go", "--yes"},
			wantErr: "ProjectName:",
		},
		{
			name:    "project name that traverses",
			args:    []string{"create", "../widget", "-l", "go", "--yes"},
			wantErr: "ProjectName:",
		},
		{
			name:    "reserved device name",
			args:    []string{"create", "nul", "-l", "go", "--yes"},
			wantErr: "ProjectName:",
		},
		{
			name:    "output directory that traverses",
			args:    []string{"create", "widget", "-l", "go", "--dir", "../../etc", "--yes"},
			wantErr: "OutputDirectory:",
		},
		{
			name:    "package manager from another ecosystem",
			args:    []string{"create", "widget", "-l", "go", "--package-manager", "npm", "--yes"},
			wantErr: "does not support package manager",
		},
		{
			name:    "malformed set option",
			args:    []string{"create", "widget", "-l", "go", "--set", "novalue", "--yes"},
			wantErr: "expected key=value",
		},
		{
			name:    "too many positional arguments",
			args:    []string{"create", "widget", "extra", "--yes"},
			wantErr: "accepts at most 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(t, nil, tt.args...)
			if err == nil {
				t.Fatalf("create = nil error, want one containing %q\n%s", tt.wantErr, out)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("create error = %v, want one containing %q", err, tt.wantErr)
			}
			if strings.Contains(out, "Configuration accepted.") {
				t.Error("an invalid configuration was accepted")
			}
			// Ordering matters: validation runs before anything is displayed,
			// so a rejected configuration never reaches the summary.
			if strings.Contains(out, summaryTitleText) {
				t.Errorf("a rejected configuration was printed as a summary\n%s", out)
			}
		})
	}
}

func TestCreateRejectsAnUnsafeAnswerFromAPrompt(t *testing.T) {
	// The traversal table below drives --dir. This pins the other front door:
	// the same value typed at the prompt must be rejected identically.
	for _, answer := range []string{"../../etc", `..\..\Windows`, "NUL"} {
		t.Run(answer, func(t *testing.T) {
			asker := fullyScripted()
			asker.Answers[askOutputDir] = answer

			out, _, err := run(t, asker, "create")
			if err == nil {
				t.Fatalf("create accepted %q from the prompt\n%s", answer, out)
			}
			if strings.Contains(out, summaryTitleText) {
				t.Errorf("an unsafe configuration reached the summary\n%s", out)
			}
		})
	}
}

func TestCreateValidatesBeforeAskingToConfirm(t *testing.T) {
	asker := fullyScripted()

	_, _, err := run(t, asker, "create", "acme/widget", "-l", "go")
	if err == nil {
		t.Fatal("create = nil error, want the invalid name to be rejected")
	}
	if slices.Contains(asker.Asked, askConfirm) {
		t.Error("the user was asked to confirm an invalid configuration")
	}
}

func TestCreatePlanFlagPrintsTheFullPlan(t *testing.T) {
	out, _, err := run(t, nil,
		"create", "widget",
		"-l", "go",
		"--description", "Widget control plane",
		"--set", "go_module=github.com/acme/widget",
		"--plan", "--yes",
	)
	if err != nil {
		t.Fatalf("create --plan error = %v\n%s", err, out)
	}

	for _, want := range []string{
		"Repository plan",
		"CLAUDE.md",
		".claude/agents/",
		".github/workflows/ci.yml",
		"go test ./...",
		"github.com/acme/widget",
		"Project Configuration",
		"Configuration accepted.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plan output is missing %q\n%s", want, out)
		}
	}
}

func TestCreateWithoutPlanFlagOmitsTheArtifactList(t *testing.T) {
	out, _, err := run(t, nil, "create", "widget", "-l", "go", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	if strings.Contains(out, "Repository plan") {
		t.Errorf("the full plan was printed without --plan\n%s", out)
	}
}

func TestCreateAppliesLanguageOptionDefaults(t *testing.T) {
	out, _, err := run(t, nil, "create", "my-widget", "-l", "python", "--plan", "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "my_widget") {
		t.Errorf("plan is missing the derived package name\n%s", out)
	}
}

func TestCreateWritesNothingToDisk(t *testing.T) {
	dir := t.TempDir()

	out, _, err := run(t, nil, "create", "widget", "-l", "go", "--dir", dir, "--yes")
	if err != nil {
		t.Fatalf("create error = %v\n%s", err, out)
	}
	if !strings.Contains(out, "Configuration accepted.") {
		t.Fatalf("create did not accept the configuration\n%s", out)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read target directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("create wrote %d entries into the target directory, want none", len(entries))
	}
}

// summaryValue returns the value printed for a summary label.
func summaryValue(t *testing.T, out, label string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), label+": "); ok {
			return strings.TrimSpace(after)
		}
	}
	t.Fatalf("summary has no %q line\n%s", label, out)
	return ""
}

func TestCreatePlanArtifactsFollowTheFeatureFlags(t *testing.T) {
	// The plan is printed directly above the confirmation, so it must never
	// promise a file the feature list in the same block says is off.
	tests := []struct {
		name     string
		flags    []string
		absent   []string
		stillHas []string
	}{
		{
			name:     "no claude",
			flags:    []string{"--no-claude"},
			absent:   []string{".claude/settings.json", ".claude/agents/", ".claude/commands/", "CLAUDE.md"},
			stillHas: []string{"README.md", ".github/workflows/ci.yml"},
		},
		{
			name:     "no ci",
			flags:    []string{"--no-ci"},
			absent:   []string{".github/workflows/ci.yml"},
			stillHas: []string{"README.md", ".claude/agents/"},
		},
		{
			name:     "no docs",
			flags:    []string{"--no-docs"},
			absent:   []string{"docs/architecture.md", "docs/testing.md"},
			stillHas: []string{"README.md"},
		},
		{
			name:     "everything on",
			flags:    nil,
			absent:   nil,
			stillHas: []string{"README.md", "CLAUDE.md", ".claude/agents/", ".github/workflows/ci.yml", "tests/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"create", "widget", "-l", "go", "--plan", "--yes"}, tt.flags...)
			out, _, err := run(t, nil, args...)
			if err != nil {
				t.Fatalf("create error = %v\n%s", err, out)
			}

			artifacts := artifactSection(t, out)
			for _, path := range tt.absent {
				if strings.Contains(artifacts, path) {
					t.Errorf("plan promises %q although the feature is off\n%s", path, artifacts)
				}
			}
			for _, path := range tt.stillHas {
				if !strings.Contains(artifacts, path) {
					t.Errorf("plan is missing %q\n%s", path, artifacts)
				}
			}
		})
	}
}

// artifactSection returns the Artifacts block of the plan.
func artifactSection(t *testing.T, out string) string {
	t.Helper()

	start := strings.Index(out, "Artifacts\n")
	if start < 0 {
		t.Fatalf("plan has no Artifacts section\n%s", out)
	}
	rest := out[start:]
	if end := strings.Index(rest, "(plus"); end >= 0 {
		return rest[:end]
	}
	return rest
}
