package run

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/sanitize"
)

// maxErrorMessageLength is the maximum length of a plan or apply error message.
const maxErrorMessageLength = 2048

// TruncateAndSanitizeErrorMessage truncates a plan or apply error message to maxErrorMessageLength and sanitizes it to valid UTF-8.
func TruncateAndSanitizeErrorMessage(errorMessage string) *string {
	if len(errorMessage) > maxErrorMessageLength {
		truncated := fmt.Sprintf(
			"%s...\n%s",
			sanitize.TruncateToValidUTF8(errorMessage, maxErrorMessageLength),
			ansi.Colorize("Error message has been truncated, check the logs for the full error message", ansi.Yellow),
		)
		return &truncated
	}

	sanitized := sanitize.ToValidUTF8(errorMessage)
	return &sanitized
}
