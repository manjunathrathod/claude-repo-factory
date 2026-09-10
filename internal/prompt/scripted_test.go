package prompt_test

import (
	"strings"
	"testing"

	"github.com/manjunathrathod/claude-repo-factory/internal/prompt"
)

func TestScriptedReplaysAnswers(t *testing.T) {
	asker := prompt.NewScripted(map[string]string{"Repository name": "widget"})

	got, err := asker.Input("Repository name", "help", "default")
	if err != nil {
		t.Fatalf("Input() error = %v", err)
	}
	if got != "widget" {
		t.Errorf("Input() = %q, want widget", got)
	}
	if len(asker.Asked) != 1 || asker.Asked[0] != "Repository name" {
		t.Errorf("Asked = %v, want the prompt to be recorded", asker.Asked)
	}
}

func TestScriptedFailsOnAnUnscriptedPrompt(t *testing.T) {
	asker := prompt.NewScripted(nil)

	_, err := asker.Input("Author", "help", "default")
	if err == nil || !strings.Contains(err.Error(), "no scripted answer") {
		t.Fatalf("Input() = %v, want a no-scripted-answer error", err)
	}
}

func TestScriptedUseDefaults(t *testing.T) {
	asker := &prompt.Scripted{UseDefaults: true}

	input, err := asker.Input("Author", "", "Platform Team")
	if err != nil {
		t.Fatalf("Input() error = %v", err)
	}
	if input != "Platform Team" {
		t.Errorf("Input() = %q, want the default", input)
	}

	choices := []prompt.Choice{{Value: "go", Label: "Go"}, {Value: "python", Label: "Python"}}
	selected, err := asker.Select("Language", "", choices, "python")
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selected != "python" {
		t.Errorf("Select() = %q, want the default", selected)
	}

	confirmed, err := asker.Confirm("Proceed", "", true)
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if !confirmed {
		t.Error("Confirm() = false, want the default")
	}
}

func TestScriptedSelectRejectsAnAnswerOutsideTheChoices(t *testing.T) {
	asker := prompt.NewScripted(map[string]string{"Language": "cobol"})
	choices := []prompt.Choice{{Value: "go", Label: "Go"}}

	_, err := asker.Select("Language", "", choices, "go")
	if err == nil || !strings.Contains(err.Error(), "not a choice") {
		t.Fatalf("Select() = %v, want a not-a-choice error", err)
	}
}

func TestScriptedConfirm(t *testing.T) {
	asker := prompt.NewScripted(nil)
	asker.Confirms["Overwrite"] = true

	got, err := asker.Confirm("Overwrite", "", false)
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if !got {
		t.Error("Confirm() = false, want the scripted true")
	}
}

// The interactive implementation must satisfy the interface the CLI depends
// on; this fails at compile time if the contract drifts.
var _ prompt.Asker = prompt.Survey{}
var _ prompt.Asker = (*prompt.Scripted)(nil)
