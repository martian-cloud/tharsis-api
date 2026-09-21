package builder

import (
	"encoding/json"
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"html/template"
)

// AnnouncementEmail is the email builder for broadcast announcement emails.
type AnnouncementEmail struct {
	Message  string
	Severity string
}

// announcementTemplateData is the rendered data passed to the announcement template.
type announcementTemplateData struct {
	// Message is the announcement rendered from markdown to HTML. goldmark escapes raw HTML in the
	// source by default (Unsafe is off), so admin-authored markup can't inject live HTML.
	Message  template.HTML
	Severity string
}

// Type returns the type of email builder
func (e *AnnouncementEmail) Type() EmailType {
	return AnnouncementEmailType
}

// Build returns the email HTML
func (e *AnnouncementEmail) Build(templateCtx *TemplateContext) (string, error) {
	if templateCtx == nil {
		return "", fmt.Errorf("template context is nil")
	}

	message, err := RenderMarkdownToHTML(e.Message)
	if err != nil {
		return "", fmt.Errorf("failed to render announcement markdown: %w", err)
	}

	data := &announcementTemplateData{
		Message:  message,
		Severity: e.Severity,
	}

	html, err := templateCtx.ExecuteTemplate(e.Type().TemplateFilename(), data)
	if err != nil {
		return "", fmt.Errorf("failed to execute announcement email template: %w", err)
	}
	return templateCtx.WrapInBaseTemplate(html)
}

// InitFromMsgpack populates the builder from the stored msgpack payload.
func (e *AnnouncementEmail) InitFromMsgpack(data []byte) error {
	return msgpack.Unmarshal(data, e)
}

// InitFromJSON populates the builder from JSON for the email preview tool.
func (e *AnnouncementEmail) InitFromJSON(data []byte) error {
	return json.Unmarshal(data, e)
}
