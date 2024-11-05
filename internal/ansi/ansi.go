// Package ansi contains ANSI color constants for coloring log output
package ansi

import "fmt"

// Code represents an ansi code
type Code string

// ANSI color constants for job log output
const (
	BoldRed   Code = "\033[31;1m"
	BoldGreen Code = "\033[32;1m"
	BoldCyan  Code = "\033[36;1m"
	Yellow    Code = "\033[33m"
	Reset     Code = "\033[0;m"
)

// Colorize wraps the string in the specified ansi color
func Colorize(s string, color Code) string {
	return fmt.Sprintf("%s%s%s", color, s, Reset)
}
