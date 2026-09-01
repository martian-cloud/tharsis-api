package models

// NamespaceType identifies whether a namespace is a group or workspace.
type NamespaceType string

// NamespaceType consts
const (
	NamespaceTypeGroup     NamespaceType = "GROUP"
	NamespaceTypeWorkspace NamespaceType = "WORKSPACE"
)
