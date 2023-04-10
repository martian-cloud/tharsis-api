package models

// Team represents a team of (human) users
type Team struct {
	Name           string
	Description    string
	SCIMExternalID string
	Metadata       ResourceMetadata
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *Team) ResolveMetadata(key string) (string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			val = t.Name
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (t *Team) Validate() error {
	// Verify description satisfies constraints
	return verifyValidDescription(t.Description)
}
