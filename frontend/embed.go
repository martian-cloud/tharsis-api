//go:build !noui

// Package frontend allows the compiled UI files to be embedded into the API binary.
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFS embed.FS

// DistFS returns the embedded frontend distribution files
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
