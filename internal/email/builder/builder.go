// Package builder handles building email templates
package builder

//go:generate go tool mockery --name EmailBuilder --inpackage --case underscore

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"github.com/vanng822/go-premailer/premailer"
)

const (
	baseTemplateFilename = "base.tmpl"

	defaultFooter = `
<p class="text">
	— The Tharsis team
</p>
`
)

// EmailType is a constant representing the various types of emails
type EmailType string

// EmailType constant values
const (
	FailedRunEmailType EmailType = "failed_run"
)

// EmailTypes returns a list of email types
func EmailTypes() []EmailType {
	return []EmailType{FailedRunEmailType}
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
	t, err := template.New(filename).Parse(string(html))
	if err != nil {
		panic(fmt.Errorf("failed to parse email template from embedded file %s: %w", filename, err))
	}
	templates[filename] = t
}

// EmailBuilder is an interface for building emails
type EmailBuilder interface {
	Type() EmailType
	Build(templateCtx *TemplateContext) (string, error)
	InitFromData(data []byte) error
}

type baseTemplateData struct {
	Footer template.HTML
	Body   template.HTML
}

type commonFields struct {
	FrontendURL string
}

// TemplateContext is the context for building templates
type TemplateContext struct {
	common    commonFields
	footer    template.HTML
	templates map[string]*template.Template
}

// NewTemplateContext creates a new template context
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
		common:    commonFields{FrontendURL: frontendURL},
		footer:    template.HTML(footer), // nosemgrep: gosec.G203-1
		templates: templates,
	}
}

// ExecuteTemplate executes the template and returns the html
func (t *TemplateContext) ExecuteTemplate(filename string, data interface{}) (string, error) {
	allData := map[string]interface{}{}
	allData["common"] = t.common
	allData["this"] = data

	var buf bytes.Buffer
	err := t.templates[filename].Execute(&buf, allData)
	if err != nil {
		return "", fmt.Errorf("failed to execute email template: %v", err)
	}

	return t.inlineCSS(buf.String())
}

// WrapInBaseTemplate wraps the body in the base template html
func (t *TemplateContext) WrapInBaseTemplate(body string) (string, error) {
	var buf bytes.Buffer
	err := t.templates[baseTemplateFilename].Execute(&buf, &baseTemplateData{
		Footer: template.HTML(t.footer), // nosemgrep: gosec.G203-1
		// Passing html body here is safe since this is not user input
		Body: template.HTML(body), // nosemgrep: gosec.G203-1
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute base email template: %v", err)
	}

	return t.inlineCSS(buf.String())
}

func (t *TemplateContext) inlineCSS(html string) (string, error) {
	prem, err := premailer.NewPremailerFromString(html, premailer.NewOptions())
	if err != nil {
		return "", fmt.Errorf("failed to inline email template css: %v", err)
	}
	return prem.Transform()
}
