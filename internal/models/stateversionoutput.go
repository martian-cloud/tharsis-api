package models

// StateVersionOutput represents a terraform state version output
type StateVersionOutput struct {
	Name           string
	StateVersionID string
	Metadata       ResourceMetadata
	Value          []byte
	Type           []byte
	Sensitive      bool
}

// The End.
