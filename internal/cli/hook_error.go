package cli

import (
	"fmt"

	"github.com/inhere/xenv/internal/xenv/shell"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
)

func outputHookWarningExpression(message string, err error) bool {
	if !xenvcom.InHookShell() {
		return false
	}

	shell.OutputScriptWithMessage(fmt.Sprintf("WARN: %s: %v", message, err), "")
	return true
}
