// Package email supports sending emails.
package email

//go:generate mockery --name Provider --inpackage --case underscore

import "context"

// Provider is an interface for sending emails and events
type Provider interface {
	SendMail(ctx context.Context, to []string, subject, body string) error
}

// NoopProvider is an email provider that doesn't send any emails
type NoopProvider struct{}

// SendMail is a noop
func (n *NoopProvider) SendMail(_ context.Context, _ []string, _, _ string) error {
	// Explicitly return nil
	return nil
}
