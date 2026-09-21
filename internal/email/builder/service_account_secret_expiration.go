package builder

import (
	"encoding/json"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// ServiceAccountSecretExpirationEmail is the email builder for service account secret expiration warnings.
type ServiceAccountSecretExpirationEmail struct {
	ServiceAccountName string
	ServiceAccountID   string
	GroupPath          string
	ExpiresAt          time.Time
}

// Type returns the type of email builder
func (e *ServiceAccountSecretExpirationEmail) Type() EmailType {
	return ServiceAccountSecretExpirationEmailType
}

// Build returns the email html
func (e *ServiceAccountSecretExpirationEmail) Build(templateCtx *TemplateContext) (string, error) {
	html, err := templateCtx.ExecuteTemplate(e.Type().TemplateFilename(), e)
	if err != nil {
		return "", err
	}
	return templateCtx.WrapInBaseTemplate(html)
}

// InitFromMsgpack populates the builder from the stored msgpack payload.
func (e *ServiceAccountSecretExpirationEmail) InitFromMsgpack(data []byte) error {
	return msgpack.Unmarshal(data, e)
}

// InitFromJSON populates the builder from JSON for the email preview tool.
func (e *ServiceAccountSecretExpirationEmail) InitFromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}
