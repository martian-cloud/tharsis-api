// Package ansi contains ANSI color constants for coloring log output
package ansi

import (
	"fmt"
	"regexp"
)

// Code represents an ansi code
type Code string

// ANSI color constants for job log output
const (
	Bold       Code = "\033[1m"
	BoldRed    Code = "\033[31;1m"
	BoldGreen  Code = "\033[32;1m"
	BoldCyan   Code = "\033[36;1m"
	BoldYellow Code = "\033[33;1m"
	Yellow     Code = "\033[33m"
	Reset      Code = "\033[0;m"
)

var (
	// this regex is from package https://pkg.go.dev/github.com/acarl005/stripansi, with MIT license.
	ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")
)

// Colorize wraps the string in the specified ansi color
func Colorize(s string, color Code) string {
	return fmt.Sprintf("%s%s%s", color, s, Reset)
}

// UnColorize removes the above ansi color codes from the string.
func UnColorize(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
