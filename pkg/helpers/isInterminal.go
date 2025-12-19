package helpers

import (
	"os"

	"golang.org/x/term"
)

func IsInTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
