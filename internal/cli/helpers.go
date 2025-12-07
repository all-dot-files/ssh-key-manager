package cli

import (
	"fmt"
	"os"

	"github.com/all-dot-files/ssh-key-manager/pkg/errors"
)

// HandleError handles an error with improved formatting and suggestions
func HandleError(err error) {
	if err == nil {
		return
	}

	// Check if it's an AppError
	if appErr, ok := err.(*errors.AppError); ok {
		if IsDebug() {
			fmt.Fprintf(os.Stderr, "❌ [%s] %s: %s\nCause: %v\n", appErr.Code, appErr.Op, appErr.Message, appErr.Err)
		} else {
			fmt.Fprintf(os.Stderr, "❌ Error: %s\n", appErr.Message)
		}
		
		// Print suggestion if available
		if appErr.Suggestion != "" {
			fmt.Fprintf(os.Stderr, "\n💡 Suggestion: %s\n", appErr.Suggestion)
		}
	} else {
		// Regular error
		if IsDebug() {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			fmt.Fprintln(os.Stderr, "\n💡 Use --debug flag for more details")
		}
	}
}

// WrapError wraps an error with category and message
func WrapError(err error, category string, message string) *errors.AppError {
	return errors.Wrap(err, category, "CMD", message)
}

// LogVerbose prints verbose output if verbose mode is enabled
func LogVerbose(format string, args ...interface{}) {
	if IsVerbose() {
		fmt.Fprintf(os.Stderr, "🔍 "+format+"\n", args...)
	}
}

// LogDebug prints debug output if debug mode is enabled
func LogDebug(format string, args ...interface{}) {
	if IsDebug() {
		fmt.Fprintf(os.Stderr, "🐛 DEBUG: "+format+"\n", args...)
	}
}

// Success prints a success message
func Success(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}

// Info prints an info message
func Info(format string, args ...interface{}) {
	fmt.Printf("ℹ️  "+format+"\n", args...)
}

// Warning prints a warning message
func Warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "⚠️  "+format+"\n", args...)
}

