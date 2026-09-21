// Package builder handles building email templates
package builder

//go:generate go tool mockery --name EmailBuilder --inpackage --case underscore

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/url"
	"slices"

	"github.com/vanng822/go-premailer/premailer"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	baseTemplateFilename = "base.tmpl"

	defaultFooter = `
<p class="text">
	— The Tharsis team
</p>
`
)

// RecipientTokenPlaceholder is the literal a rendered template embeds in a link back into the app;
// the sender substitutes it with the actual recipient's global ID before sending, since a template
// is rendered once and shared across all of an outbox's recipients.
const RecipientTokenPlaceholder = "__RECIPIENT_TOKEN__"

// EmailType is a constant representing the various types of emails
type EmailType string

// EmailType constant values
const (
	FailedRunEmailType                      EmailType = "failed_run"
	ServiceAccountSecretExpirationEmailType EmailType = "service_account_secret_expiration"
	SigningKeyDecommissionEmailType         EmailType = "signing_key_decommission"
	MembershipChangeEmailType               EmailType = "membership_change"
	AnnouncementEmailType                   EmailType = "announcement"
)

// EmailTypes returns a list of email types
func EmailTypes() []EmailType {
	return []EmailType{FailedRunEmailType, ServiceAccountSecretExpirationEmailType, SigningKeyDecommissionEmailType, MembershipChangeEmailType, AnnouncementEmailType}
}

// Valid reports whether et is a known email type.
func (et EmailType) Valid() bool {
	return slices.Contains(EmailTypes(), et)
}

// TemplateFilename returns the template filename for this type
func (et EmailType) TemplateFilename() string {
	return fmt.Sprintf("%s.tmpl", et)
}

// NewBuilder returns a new builder
func (et EmailType) NewBuilder() (EmailBuilder, error) {
	switch et {
	case FailedRunEmailType:
		return &FailedRunEmail{}, nil
	case ServiceAccountSecretExpirationEmailType:
		return &ServiceAccountSecretExpirationEmail{}, nil
	case SigningKeyDecommissionEmailType:
		return &SigningKeyDecommissionEmail{}, nil
	case MembershipChangeEmailType:
		return &MembershipChangeEmail{}, nil
	case AnnouncementEmailType:
		return &AnnouncementEmail{}, nil
	default:
		return nil, fmt.Errorf("unknown email type: %s", et)
	}
}

//go:embed templates
var embedFS embed.FS

func registerTemplate(filename string, templates map[string]*template.Template) {
	html, err := embedFS.ReadFile("templates/" + filename)
	if err != nil {
		panic(fmt.Errorf("failed to read email template from embedded file %s: %w", filename, err))
	}
	t, err := template.New(filename).Funcs(template.FuncMap{"pathEscape": url.PathEscape}).Parse(string(html))
	if err != nil {
		panic(fmt.Errorf("failed to parse email template from embedded file %s: %w", filename, err))
	}
	templates[filename] = t
}

// EmailBuilder is an interface for building emails
type EmailBuilder interface {
	// Type returns the email type, which is also the outbox email_type and template name.
	Type() EmailType
	// Build renders the email body HTML using the given template context.
	Build(templateCtx *TemplateContext) (string, error)
	// InitFromMsgpack populates the builder from the outbox's stored payload, which the enqueuer
	// serializes with msgpack. This is the path the sender uses.
	InitFromMsgpack(data []byte) error
	// InitFromJSON populates the builder from JSON. The email preview tool feeds it hand-authored
	// JSON from an env var, so preview is the only caller.
	InitFromJSON(data []byte) error
}

type baseTemplateData struct {
	Footer template.HTML
	Body   template.HTML
}

type commonFields struct {
	FrontendURL               string
	RecipientTokenPlaceholder string
}

// TemplateContext is the context for building templates
type TemplateContext struct {
	common    commonFields
	footer    template.HTML
	templates map[string]*template.Template
}

// NewTemplateContext creates a new template context.
func NewTemplateContext(frontendURL string, footer string) *TemplateContext {
	if footer == "" {
		footer = defaultFooter
	}

	templates := make(map[string]*template.Template)

	registerTemplate(baseTemplateFilename, templates)
	for _, et := range EmailTypes() {
		registerTemplate(et.TemplateFilename(), templates)
	}

	return &TemplateContext{
		common:    commonFields{FrontendURL: frontendURL, RecipientTokenPlaceholder: RecipientTokenPlaceholder},
		footer:    template.HTML(footer), // nosemgrep: gosec.G203-1
		templates: templates,
	}
}

// ExecuteTemplate executes the template and returns the html; CSS is inlined once later by WrapInBaseTemplate.
func (t *TemplateContext) ExecuteTemplate(filename string, data interface{}) (string, error) {
	allData := map[string]interface{}{}
	allData["common"] = t.common
	allData["this"] = data

	var buf bytes.Buffer
	err := t.templates[filename].Execute(&buf, allData)
	if err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// WrapInBaseTemplate wraps the body in the base template html
func (t *TemplateContext) WrapInBaseTemplate(body string) (string, error) {
	var buf bytes.Buffer
	err := t.templates[baseTemplateFilename].Execute(&buf, &baseTemplateData{
		Footer: template.HTML(t.footer), // nosemgrep: gosec.G203-1
		Body:   template.HTML(body),     // nosemgrep: gosec.G203-1
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute base email template: %w", err)
	}

	return t.inlineCSS(buf.String())
}

func (t *TemplateContext) inlineCSS(html string) (string, error) {
	prem, err := premailer.NewPremailerFromString(html, premailer.NewOptions())
	if err != nil {
		return "", fmt.Errorf("failed to inline email template css: %w", err)
	}
	return prem.Transform()
}

// RenderMarkdownToHTML renders author-supplied markdown to HTML with raw HTML escaped (Unsafe off) and single newlines hard-wrapped to <br>; block spacing comes from the email CSS.
func RenderMarkdownToHTML(source string) (template.HTML, error) {
	var rendered bytes.Buffer
	md := goldmark.New(goldmark.WithRendererOptions(html.WithHardWraps()))
	if err := md.Convert([]byte(source), &rendered); err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	return template.HTML(rendered.String()), nil // nosemgrep: gosec.G203-1
}
