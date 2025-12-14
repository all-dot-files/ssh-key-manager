package integration

import (
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
)

func TestSuggestionsForUnknownCommand(t *testing.T) {
	cmd := cli.RootCommandForTest()
	suggestions := cmd.SuggestionsFor("kye")
	if len(suggestions) == 0 {
		t.Fatalf("expected suggestions for mistyped command")
	}
}
