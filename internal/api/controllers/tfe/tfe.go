package tfe

import "strings"

// convertOrgToGroupPath takes an organization path and converts it to a tharsis group
func convertOrgToGroupPath(org string) string {
	parts := strings.Split(org, ".")
	return strings.Join(parts, "/")
}
