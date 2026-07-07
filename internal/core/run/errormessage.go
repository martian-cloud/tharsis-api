package run

import (
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
)

// maxErrorMessageLength is the maximum length of a plan or apply error message.
const maxErrorMessageLength = 2048

// SanitizeAndTruncateErrorMessage sanitizes UTF-8 characters and truncates if needed.
func SanitizeAndTruncateErrorMessage(errorMessage string) *string {
	// First sanitize UTF-8 - replace invalid sequences with replacement character
	sanitized := strings.ToValidUTF8(errorMessage, "�")

	if len(sanitized) > maxErrorMessageLength {
		truncated := fmt.Sprintf(
			"%s...\n%s",
			sanitized[:maxErrorMessageLength],
			ansi.Colorize("Error message has been truncated, check the logs for the full error message", ansi.Yellow),
		)
		return &truncated
	}
	return &sanitized
}
