package run

import (
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
)

// maxErrorMessageLength is the maximum length of a plan or apply error message.
const maxErrorMessageLength = 2048

// TruncateAndSanitizeErrorMessage truncates a plan or apply error message to maxErrorMessageLength
// and sanitizes it to valid UTF-8.
func TruncateAndSanitizeErrorMessage(errorMessage string) *string {
	if len(errorMessage) > maxErrorMessageLength {
		slice := errorMessage[:maxErrorMessageLength]
		sanitized := strings.ToValidUTF8(slice, "�")
		truncated := fmt.Sprintf(
			"%s...\n%s",
			sanitized,
			ansi.Colorize("Error message has been truncated, check the logs for the full error message", ansi.Yellow),
		)
		return &truncated
	}

	sanitized := strings.ToValidUTF8(errorMessage, "�")
	return &sanitized
}
