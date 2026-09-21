package builder

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"
)

// RunStage indicates which stage (plan or apply) errored.
type RunStage string

const (
	// PlanStage is the plan stage.
	PlanStage RunStage = "plan"
	// ApplyStage is the apply stage.
	ApplyStage RunStage = "apply"
)

// FailedRunEmail is the email builder for failed runs reports.
type FailedRunEmail struct {
	WorkspacePath string
	Title         string
	ModuleVersion *string
	ModuleSource  *string
	CreatedBy     string
	ErrorMessage  string
	RunID         string
	RunStage      RunStage
}

type failedRunEmailTemplateData struct {
	FailedRunEmail
}

// Type returns the type of email builder
func (fr *FailedRunEmail) Type() EmailType {
	return FailedRunEmailType
}

// Build returns the email html
func (fr *FailedRunEmail) Build(templateCtx *TemplateContext) (string, error) {
	templateData := &failedRunEmailTemplateData{
		FailedRunEmail: *fr,
	}

	html, err := templateCtx.ExecuteTemplate(fr.Type().TemplateFilename(), templateData)
	if err != nil {
		return "", err
	}
	return templateCtx.WrapInBaseTemplate(html)
}

// InitFromMsgpack populates the builder from the stored msgpack payload.
func (fr *FailedRunEmail) InitFromMsgpack(data []byte) error {
	return msgpack.Unmarshal(data, fr)
}

// InitFromJSON populates the builder from JSON for the email preview tool.
func (fr *FailedRunEmail) InitFromJSON(data []byte) error {
	return json.Unmarshal(data, fr)
}
