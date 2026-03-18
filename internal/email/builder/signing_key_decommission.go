package builder

import (
	"encoding/json"
	"fmt"
	"time"
)

// SigningKeyDecommissionEmail is the email builder for signing key decommission alerts
type SigningKeyDecommissionEmail struct {
	KeyID                    string
	DecommissioningStartedAt time.Time
	DeletionTime             time.Time
}

// Type returns the type of email builder
func (e *SigningKeyDecommissionEmail) Type() EmailType {
	return SigningKeyDecommissionEmailType
}

// Build returns the email HTML
func (e *SigningKeyDecommissionEmail) Build(templateCtx *TemplateContext) (string, error) {
	if templateCtx == nil {
		return "", fmt.Errorf("template context is nil")
	}
	html, err := templateCtx.ExecuteTemplate(e.Type().TemplateFilename(), e)
	if err != nil {
		return "", fmt.Errorf("failed to execute signing key decommission email template: %v", err)
	}
	return templateCtx.WrapInBaseTemplate(html)
}

// InitFromData creates the builder from raw data
func (e *SigningKeyDecommissionEmail) InitFromData(data []byte) error {
	return json.Unmarshal(data, e)
}
