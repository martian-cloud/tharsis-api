package models

// Team represents a team of (human) users
type Team struct {
	Name           string
	Description    string
	SCIMExternalID string
	Metadata       ResourceMetadata
}

// Validate returns an error if the model is not valid
func (t *Team) Validate() error {
	// Verify description satisfies constraints
	if err := verifyValidDescription(t.Description); err != nil {
		return err
	}
	return nil
}

// The End.
