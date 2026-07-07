package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

	graphql "github.com/graph-gophers/graphql-go"
)

/* Apply Query Resolvers */

// ApplyResolver resolves a apply resource
type ApplyResolver struct {
	run *models.Run
}

// ID resolver
func (r *ApplyResolver) ID() graphql.ID {
	return graphql.ID(r.run.Apply.GetGlobalID())
}

// Status resolver
func (r *ApplyResolver) Status() string {
	return string(r.run.Apply.Status)
}

// ErrorMessage resolver
func (r *ApplyResolver) ErrorMessage() *string {
	return r.run.Apply.ErrorMessage
}

// TriggeredBy resolver
func (r *ApplyResolver) TriggeredBy() *string {
	apply := r.run.Apply
	if apply.TriggeredBy == "" {
		return nil
	}
	return &apply.TriggeredBy
}

// Metadata resolver
func (r *ApplyResolver) Metadata() (*MetadataResolver, error) {
	apply := r.run.Apply
	return &MetadataResolver{metadata: apply.Metadata(r.run)}, nil
}

// Jobs returns the connection of jobs associated with the apply.
func (r *ApplyResolver) Jobs(ctx context.Context, args *ConnectionQueryArgs) (*JobConnectionResolver, error) {
	return jobConnectionForRunNode(ctx, r.run, models.JobApplyType, args)
}

// CurrentJob returns the current job for the apply resource
func (r *ApplyResolver) CurrentJob(ctx context.Context) (*JobResolver, error) {
	apply := r.run.Apply
	if apply.LatestJobID == nil {
		return nil, nil
	}

	job, err := loadJob(ctx, *apply.LatestJobID)
	if err != nil {
		return nil, err
	}

	return &JobResolver{job: job}, nil
}

// Comment resolver
func (r *ApplyResolver) Comment() string {
	return r.run.Apply.Comment
}
