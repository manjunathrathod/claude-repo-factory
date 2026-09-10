// Package prompt isolates interactive questioning behind a small interface.
//
// The CLI depends on Asker, never on survey directly. That keeps the command
// layer testable without a TTY: tests inject a Scripted asker, and the
// interactive implementation is the only code that needs a terminal.
package prompt

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// ErrInterrupted is returned when the user aborts a prompt, typically with
// Ctrl+C. Callers should exit quietly rather than reporting a failure.
var ErrInterrupted = errors.New("prompt: cancelled by user")

// Choice is a selectable option with a stable value and a human label.
type Choice struct {
	Value string
	Label string
}

// Asker collects answers from the user.
type Asker interface {
	Input(message, help, def string) (string, error)
	Select(message, help string, choices []Choice, def string) (string, error)
	Confirm(message, help string, def bool) (bool, error)
}

// Survey is the interactive Asker backed by the survey library.
type Survey struct{}

// Input implements Asker.
func (Survey) Input(message, help, def string) (string, error) {
	var answer string
	q := &survey.Input{Message: message, Help: help, Default: def}
	if err := survey.AskOne(q, &answer, survey.WithValidator(survey.Required)); err != nil {
		return "", wrap(err)
	}
	return answer, nil
}

// Select implements Asker.
func (Survey) Select(message, help string, choices []Choice, def string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("prompt: %s has no choices", message)
	}
	labels := make([]string, 0, len(choices))
	byLabel := make(map[string]string, len(choices))
	defLabel := choices[0].Label
	for _, c := range choices {
		labels = append(labels, c.Label)
		byLabel[c.Label] = c.Value
		if c.Value == def {
			defLabel = c.Label
		}
	}

	var picked string
	q := &survey.Select{Message: message, Help: help, Options: labels, Default: defLabel}
	if err := survey.AskOne(q, &picked); err != nil {
		return "", wrap(err)
	}
	return byLabel[picked], nil
}

// Confirm implements Asker.
func (Survey) Confirm(message, help string, def bool) (bool, error) {
	var answer bool
	q := &survey.Confirm{Message: message, Help: help, Default: def}
	if err := survey.AskOne(q, &answer); err != nil {
		return false, wrap(err)
	}
	return answer, nil
}

func wrap(err error) error {
	if errors.Is(err, terminalInterrupt) {
		return ErrInterrupted
	}
	return err
}
