package plan

//go:generate go tool mockery --name Parser --inpackage --case underscore

import (
	"fmt"
	"sort"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"

	tjson "github.com/hashicorp/terraform-json"
)

// Diff is a model for a normalized diff
type Diff struct {
	Resources []*ResourceDiff `json:"resources"`
	Outputs   []*OutputDiff   `json:"outputs"`
}

// ChangeWarning describes a warning that occurred during a plan
type ChangeWarning struct {
	ChangeType string `json:"change_type"`
	Message    string `json:"message"`
	Line       int32  `json:"line"`
}

// OutputDiff is a model for an output diff
type OutputDiff struct {
	Action         action.Action    `json:"action"`
	OutputName     string           `json:"output_name"`
	UnifiedDiff    string           `json:"unified_diff"`
	OriginalSource string           `json:"original_source"`
	Warnings       []*ChangeWarning `json:"warnings"`
}

// ResourceDiff is a model for a resource diff
type ResourceDiff struct {
	Mode           string           `json:"mode"`
	Address        string           `json:"address"`
	ResourceType   string           `json:"resource_type"`
	ResourceName   string           `json:"resource_name"`
	ProviderName   string           `json:"provider_name"`
	ModuleAddress  string           `json:"module_address"`
	Action         action.Action    `json:"action"`
	UnifiedDiff    string           `json:"unified_diff"`
	OriginalSource string           `json:"original_source"`
	Warnings       []*ChangeWarning `json:"warnings"`
	Imported       bool             `json:"imported"`
	Drifted        bool             `json:"drifted"`
	Moved          bool             `json:"moved"`
}

// Parser is used to extract a normalized diff from a terraform plan
type Parser interface {
	Parse(plan *tjson.Plan, schemas *tjson.ProviderSchemas) (*Diff, error)
}

type parser struct{}

// NewParser creates a new parser for the given plan and provider schemas
func NewParser() Parser {
	return &parser{}
}

// Parse parses the plan and returns the normalized diff
func (p *parser) Parse(plan *tjson.Plan, schemas *tjson.ProviderSchemas) (*Diff, error) {
	outputDiffs := []*OutputDiff{}
	resourceDiffs := []*ResourceDiff{}

	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("the plan json format is not valid: %w", err)
	}

	if err := schemas.Validate(); err != nil {
		return nil, fmt.Errorf("the provider schemas json is not valid: %w", err)
	}

	rawDiffs, err := p.precomputeDiffs(plan, schemas)
	if err != nil {
		return nil, err
	}

	var keys []string
	for key := range rawDiffs.outputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		outputDiff, err := rawDiffs.outputs[key].decode()
		if err != nil {
			return nil, err
		}

		if outputDiff.Action == action.NoOp {
			continue
		}

		outputDiffs = append(outputDiffs, outputDiff)
	}

	for _, change := range rawDiffs.changes {
		resourceDiff, err := change.decode()
		if err != nil {
			return nil, err
		}

		if resourceDiff.Action == action.NoOp && !resourceDiff.Moved && !resourceDiff.Imported {
			// Don't show anything for NoOp changes unless they are moved or imported
			continue
		}

		resourceDiffs = append(resourceDiffs, resourceDiff)
	}

	return &Diff{
		Resources: resourceDiffs,
		Outputs:   outputDiffs,
	}, nil
}
