package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/all-dot-files/ssh-key-manager/pkg/errors"
)

// PrintError prints the error in a user-friendly format
func PrintError(err error) {
	if err == nil {
		return
	}

	// Colors
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.FgHiBlack).SprintFunc()

	// Check if it's an AppError
	if appErr, ok := err.(*errors.AppError); ok {
		fmt.Fprintf(os.Stderr, "%s %s\n", red("Wait!"), appErr.Message)

		if appErr.Suggestion != "" {
			fmt.Fprintf(os.Stderr, "\n%s %s\n", yellow("Suggestion:"), appErr.Suggestion)
		}

		if IsDebug() {
			fmt.Fprintf(os.Stderr, "\n%s\n", dim("--- Debug Info ---"))
			fmt.Fprintf(os.Stderr, "%s Code: %s\n", dim("•"), appErr.Code)
			fmt.Fprintf(os.Stderr, "%s Op:   %s\n", dim("•"), appErr.Op)
			if appErr.Err != nil {
				fmt.Fprintf(os.Stderr, "%s Cause: %+v\n", dim("•"), appErr.Err)
			}
		}
	} else {
		// Generic error
		fmt.Fprintf(os.Stderr, "%s %s\n", red("Error:"), err.Error())

		if IsDebug() {
			fmt.Fprintf(os.Stderr, "\n%s %+v\n", dim("Debug:"), err)
		}
	}
}

// ExitWithError prints the error and exits with code 1
func ExitWithError(err error) {
	PrintError(err)
	os.Exit(1)
}
