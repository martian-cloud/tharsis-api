// Package secret contains the secret manager interface which is used to manage secret values.
package secret

import "context"

//go:generate go tool mockery --name Manager --inpackage --case underscore

// Manager manages secret values
type Manager interface {
	Create(ctx context.Context, key string, value string) ([]byte, error)
	Update(ctx context.Context, key string, secretData []byte, newValue string) ([]byte, error)
	Get(ctx context.Context, key string, secretData []byte) (string, error)
}

// NoopManager is a secret manager that does nothing
type NoopManager struct{}

// Create does nothing and returns an empty string and nil error
func (m *NoopManager) Create(_ context.Context, _ string, value string) ([]byte, error) {
	return []byte(value), nil
}

// Update does nothing and returns an empty string and nil error
func (m *NoopManager) Update(_ context.Context, _ string, _ []byte, newValue string) ([]byte, error) {
	return []byte(newValue), nil
}

// Get does nothing and returns an empty string and nil error
func (m *NoopManager) Get(_ context.Context, _ string, secretData []byte) (string, error) {
	return string(secretData), nil
}
