package prompt

import "fmt"

// Scripted is a non-interactive Asker that replays pre-recorded answers.
// It exists for tests and for headless runs, and it fails loudly when a
// prompt has no scripted answer so a test can never silently accept a
// default it did not intend.
type Scripted struct {
	// Answers maps a prompt message to the answer to return.
	Answers map[string]string
	// Confirms maps a prompt message to a boolean answer.
	Confirms map[string]bool
	// UseDefaults makes unscripted prompts return their default instead of
	// failing.
	UseDefaults bool
	// Asked records every prompt message in the order it was presented.
	Asked []string
}

// NewScripted returns a Scripted asker with the given answers.
func NewScripted(answers map[string]string) *Scripted {
	return &Scripted{Answers: answers, Confirms: map[string]bool{}}
}

// Input implements Asker.
func (s *Scripted) Input(message, _, def string) (string, error) {
	s.Asked = append(s.Asked, message)
	if v, ok := s.Answers[message]; ok {
		return v, nil
	}
	if s.UseDefaults {
		return def, nil
	}
	return "", fmt.Errorf("prompt: no scripted answer for %q", message)
}

// Select implements Asker.
func (s *Scripted) Select(message, _ string, choices []Choice, def string) (string, error) {
	s.Asked = append(s.Asked, message)
	if v, ok := s.Answers[message]; ok {
		for _, c := range choices {
			if c.Value == v {
				return v, nil
			}
		}
		return "", fmt.Errorf("prompt: scripted answer %q is not a choice for %q", v, message)
	}
	if s.UseDefaults {
		return def, nil
	}
	return "", fmt.Errorf("prompt: no scripted answer for %q", message)
}

// Confirm implements Asker.
func (s *Scripted) Confirm(message, _ string, def bool) (bool, error) {
	s.Asked = append(s.Asked, message)
	if v, ok := s.Confirms[message]; ok {
		return v, nil
	}
	if s.UseDefaults {
		return def, nil
	}
	return false, fmt.Errorf("prompt: no scripted answer for %q", message)
}
