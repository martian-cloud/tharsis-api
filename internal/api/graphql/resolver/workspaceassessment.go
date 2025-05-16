package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* WorkspaceAssessment Query Resolvers */

// WorkspaceAssessmentResolver resolves a assessment resource
type WorkspaceAssessmentResolver struct {
	assessment *models.WorkspaceAssessment
}

// ID resolver
func (r *WorkspaceAssessmentResolver) ID() graphql.ID {
	return graphql.ID(r.assessment.GetGlobalID())
}

// Metadata resolver
func (r *WorkspaceAssessmentResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.assessment.Metadata}
}

// StartedAt resolver
func (r *WorkspaceAssessmentResolver) StartedAt() graphql.Time {
	return graphql.Time{Time: r.assessment.StartedAtTimestamp}
}

// CompletedAt resolver
func (r *WorkspaceAssessmentResolver) CompletedAt() *graphql.Time {
	if r.assessment.CompletedAtTimestamp == nil {
		return nil
	}
	return &graphql.Time{Time: *r.assessment.CompletedAtTimestamp}
}

// Run resolver
func (r *WorkspaceAssessmentResolver) Run(ctx context.Context) (*RunResolver, error) {
	if r.assessment.RunID == nil {
		return nil, nil
	}
	run, err := loadRun(ctx, *r.assessment.RunID)
	if err != nil {
		return nil, err
	}

	return &RunResolver{run: run}, nil
}

// HasDrift resolver
func (r *WorkspaceAssessmentResolver) HasDrift() bool {
	return r.assessment.HasDrift
}

/* WorkspaceAssessment loader */

const assessmentLoaderKey = "assessment"

// RegisterWorkspaceAssessmentLoader registers a assessment loader function
func RegisterWorkspaceAssessmentLoader(collection *loader.Collection) {
	collection.Register(assessmentLoaderKey, assessmentBatchFunc)
}

func loadWorkspaceAssessment(ctx context.Context, workspaceID string) (*models.WorkspaceAssessment, error) {
	ldr, err := loader.Extract(ctx, assessmentLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(workspaceID))()
	if err != nil {
		return nil, err
	}

	ws, ok := data.(models.WorkspaceAssessment)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &ws, nil
}

func assessmentBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	assessments, err := getServiceCatalog(ctx).WorkspaceService.GetWorkspaceAssessmentsByWorkspaceIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range assessments {
		batch[result.WorkspaceID] = result
	}

	return batch, nil
}
