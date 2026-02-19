package builder

import (
	"encoding/json"
	"time"
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

// InitFromData creates the builder from raw data
func (e *ServiceAccountSecretExpirationEmail) InitFromData(data []byte) error {
	return json.Unmarshal(data, e)
}
