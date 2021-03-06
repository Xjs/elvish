package cli

import (
	"github.com/elves/elvish/cli/clitypes"
	"github.com/elves/elvish/cli/cliutil"
	"github.com/elves/elvish/cli/term"
)

// CommitEOF is an EventHandler that calls CommitEOF.
func CommitEOF(ev KeyEvent) { ev.CommitEOF() }

// CommitCode is an EventHandler that calls CommitCode.
func CommitCode(ev KeyEvent) { ev.CommitCode() }

// DefaultInsert is an EventHandler that is suitable as the default EventHandler
// of insert mode.
func DefaultInsert(ev KeyEvent) {
	action := cliutil.BasicHandler(term.KeyEvent(ev.Key()), ev.State())
	switch action {
	case clitypes.CommitCode:
		ev.CommitCode()
	case clitypes.CommitEOF:
		ev.CommitEOF()
	}
}

// ResetMode is an EventHandler that resets the App to its default mode.
func ResetMode(ev KeyEvent) {
	ev.State().SetMode(nil)
}
