// Package plan contains the logic for parsing a Terraform plan into a normalized diff
package plan

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured/attributepath"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/visitor"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/differ"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	tjson "github.com/hashicorp/terraform-json"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

func (p *parser) getSchema(change *tjson.ResourceChange, schemas *tjson.ProviderSchemas) (*tjson.Schema, error) {
	switch change.Mode {
	case tjson.ManagedResourceMode:
		return schemas.Schemas[change.ProviderName].ResourceSchemas[change.Type], nil
	case tjson.DataResourceMode:
		return schemas.Schemas[change.ProviderName].DataSourceSchemas[change.Type], nil
	default:
		return nil, fmt.Errorf("found unrecognized resource mode: %s", change.Mode)
	}
}

func (p *parser) precomputeDiffs(plan *tjson.Plan, schemas *tjson.ProviderSchemas) (*rawPlanDiffs, error) {
	changeAddressMap := map[string]struct{}{}
	driftAddressMap := map[string]struct{}{}

	for _, drift := range plan.ResourceDrift {
		driftAddressMap[drift.Address] = struct{}{}
	}

	diffs := rawPlanDiffs{
		outputs: make(map[string]rawOutputDiff),
	}

	for _, change := range plan.ResourceChanges {
		schema, err := p.getSchema(change, schemas)
		if err != nil {
			return nil, err
		}
		structuredChange, err := structured.FromJSONChange(*change.Change, attributepath.AlwaysMatcher())
		if err != nil {
			return nil, err
		}
		blockDiff, err := differ.ComputeDiffForBlock(structuredChange, schema.Block)
		if err != nil {
			return nil, err
		}
		_, drifted := driftAddressMap[change.Address]
		diffs.changes = append(diffs.changes, rawResourceDiff{
			change:  *change,
			diff:    blockDiff,
			drifted: drifted,
		})
		changeAddressMap[change.Address] = struct{}{}
	}

	for _, drift := range plan.ResourceDrift {
		if _, ok := changeAddressMap[drift.Address]; ok {
			// Skip drift changes that are also in the changes list.
			continue
		}

		schema, err := p.getSchema(drift, schemas)
		if err != nil {
			return nil, err
		}
		change, err := structured.FromJSONChange(*drift.Change, attributepath.AlwaysMatcher())
		if err != nil {
			return nil, err
		}

		blockDiff, err := differ.ComputeDiffForBlock(change, schema.Block)
		if err != nil {
			return nil, err
		}
		diffs.changes = append(diffs.changes, rawResourceDiff{
			change:  *drift,
			diff:    blockDiff,
			drifted: true,
		})
	}

	for key, output := range plan.OutputChanges {
		change, err := structured.FromJSONChange(*output, attributepath.AlwaysMatcher())
		if err != nil {
			return nil, err
		}
		outputDiff, err := differ.ComputeDiffForOutput(change)
		if err != nil {
			return nil, err
		}
		diffs.outputs[key] = rawOutputDiff{key: key, diff: outputDiff}
	}

	return &diffs, nil
}

type rawPlanDiffs struct {
	changes []rawResourceDiff
	outputs map[string]rawOutputDiff
}

type rawOutputDiff struct {
	key  string
	diff computed.Diff
}

func (r rawOutputDiff) decode() (*OutputDiff, error) {
	renderedDiff, err := r.diff.Render()
	if err != nil {
		return nil, err
	}

	beforeVisitor := visitor.NewBeforeVisitor(1)
	renderedDiff.Accept(beforeVisitor)

	afterVisitor := visitor.NewAfterVisitor(1)
	renderedDiff.Accept(afterVisitor)

	warnings := []*ChangeWarning{}
	for _, warning := range beforeVisitor.Warnings() {
		warnings = append(warnings, &ChangeWarning{Line: int32(warning.Line), ChangeType: "before", Message: warning.Message})
	}

	for _, warning := range afterVisitor.Warnings() {
		warnings = append(warnings, &ChangeWarning{Line: int32(warning.Line), ChangeType: "after", Message: warning.Message})
	}

	var beforeHCL, afterHCL string
	switch r.diff.Action {
	case action.Create:
		afterHCL = fmt.Sprintf("output %q {\n   value = %s\n}", r.key, afterVisitor.String())
	case action.Delete:
		beforeHCL = fmt.Sprintf("output %q {\n   value = %s\n}", r.key, beforeVisitor.String())
	default:
		beforeHCL = fmt.Sprintf("output %q {\n   value = %s\n}", r.key, beforeVisitor.String())
		afterHCL = fmt.Sprintf("output %q {\n   value = %s\n}", r.key, afterVisitor.String())
	}

	edits := myers.ComputeEdits(span.URIFromPath("before"), beforeHCL, afterHCL)
	unifiedDiff := gotextdiff.ToUnified("before", "after", beforeHCL, edits)

	diffStr := fmt.Sprint(unifiedDiff)

	return &OutputDiff{
		Action:         r.diff.Action,
		OutputName:     r.key,
		UnifiedDiff:    diffStr,
		OriginalSource: beforeHCL,
		Warnings:       warnings,
	}, nil
}

type rawResourceDiff struct {
	change  tjson.ResourceChange
	diff    computed.Diff
	drifted bool
}

func (r rawResourceDiff) Moved() bool {
	return len(r.change.PreviousAddress) > 0 && r.change.PreviousAddress != r.change.Address
}

func (r rawResourceDiff) Importing() bool {
	return r.change.Change.Importing != nil
}

func (r rawResourceDiff) Action() (action.Action, error) {
	return action.UnmarshalActions(r.change.Change.Actions)
}

func (r rawResourceDiff) decode() (*ResourceDiff, error) {
	block := "resource"
	if r.change.Mode == tjson.DataResourceMode {
		block = "data"
	}

	renderedNode, err := r.diff.Render()
	if err != nil {
		return nil, err
	}

	// Create a visitor to render the diff
	beforeVisitor := visitor.NewBeforeVisitor(0)
	renderedNode.Accept(beforeVisitor)

	afterVisitor := visitor.NewAfterVisitor(0)
	renderedNode.Accept(afterVisitor)

	warnings := []*ChangeWarning{}
	for _, warning := range beforeVisitor.Warnings() {
		warnings = append(warnings, &ChangeWarning{Line: int32(warning.Line), ChangeType: "before", Message: warning.Message})
	}

	for _, warning := range afterVisitor.Warnings() {
		warnings = append(warnings, &ChangeWarning{Line: int32(warning.Line), ChangeType: "after", Message: warning.Message})
	}

	actionType, err := r.Action()
	if err != nil {
		return nil, err
	}

	var beforeHCL, afterHCL string
	switch actionType {
	case action.Create:
		afterHCL = fmt.Sprintf("%s %q %q %s", block, r.change.Type, r.change.Name, afterVisitor.String())
	case action.Delete:
		beforeHCL = fmt.Sprintf("%s %q %q %s", block, r.change.Type, r.change.Name, beforeVisitor.String())
	default:
		beforeHCL = fmt.Sprintf("%s %q %q %s", block, r.change.Type, r.change.Name, beforeVisitor.String())
		afterHCL = fmt.Sprintf("%s %q %q %s", block, r.change.Type, r.change.Name, afterVisitor.String())
	}

	edits := myers.ComputeEdits(span.URIFromPath("before"), beforeHCL, afterHCL)
	unifiedDiff := gotextdiff.ToUnified("before", "after", beforeHCL, edits)

	return &ResourceDiff{
		Action:         actionType,
		Mode:           string(r.change.Mode),
		Address:        r.change.Address,
		ResourceType:   r.change.Type,
		ResourceName:   r.change.Name,
		ProviderName:   r.change.ProviderName,
		ModuleAddress:  r.change.ModuleAddress,
		UnifiedDiff:    fmt.Sprint(unifiedDiff),
		OriginalSource: beforeHCL,
		Imported:       r.Importing(),
		Moved:          r.Moved(),
		Drifted:        r.drifted,
		Warnings:       warnings,
	}, nil
}
