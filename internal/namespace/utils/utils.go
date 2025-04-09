// Package utils provides utility functions for working with namespaces
package utils

import "strings"

// ExpandPath takes a namespace path and returns all paths from the root to the given path.
func ExpandPath(path string) []string {
	pathParts := strings.Split(path, "/")

	paths := []string{}
	for len(pathParts) > 0 {
		paths = append(paths, strings.Join(pathParts, "/"))
		// Remove last element
		pathParts = pathParts[:len(pathParts)-1]
	}

	return paths
}

// IsDescendantOfPath returns true if the namespace is a descendant of the specified (ancestor group) path.
func IsDescendantOfPath(descendantPath, ancestorPath string) bool {
	return strings.HasPrefix(descendantPath, ancestorPath+"/")
}

