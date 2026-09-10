package prompt

import "github.com/AlecAivazis/survey/v2/terminal"

// terminalInterrupt is the sentinel survey returns when the user presses
// Ctrl+C. It is isolated here so the rest of the package does not depend on
// the survey terminal package.
var terminalInterrupt = terminal.InterruptErr
