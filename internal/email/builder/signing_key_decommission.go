package builder

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
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

// InitFromMsgpack populates the builder from the stored msgpack payload.
func (e *SigningKeyDecommissionEmail) InitFromMsgpack(data []byte) error {
	return msgpack.Unmarshal(data, e)
}

// InitFromJSON populates the builder from JSON for the email preview tool.
func (e *SigningKeyDecommissionEmail) InitFromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}
