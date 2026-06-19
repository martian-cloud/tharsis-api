// Package adminlogtail contains the service for the admin log tail viewer.
package adminlogtail

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger/logstore"
)

// GetEntriesInput is the input for GetEntries.
type GetEntriesInput struct {
	Levels []string
	Search string
	Limit  int
}

// Service is the interface for the admin log tail service.
type Service interface {
	GetEntries(ctx context.Context, input *GetEntriesInput) ([]*logstore.LogEntry, error)
	Subscribe(ctx context.Context) (<-chan *logstore.LogEntry, error)
}

type service struct {
	store logstore.Store
}

// NewService creates a new admin log tail service.
func NewService(store logstore.Store) Service {
	return &service{store: store}
}

func (s *service) GetEntries(ctx context.Context, input *GetEntriesInput) ([]*logstore.LogEntry, error) {
	ctx, span := tracer.Start(ctx, "svc.GetEntries")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if !caller.IsAdminModeActivated(ctx) {
		return nil, errors.New("only admins with admin mode activated can view admin log tail entries", errors.WithErrorCode(errors.EForbidden))
	}

	return s.store.GetEntries(input.Levels, input.Search, input.Limit)
}

func (s *service) Subscribe(ctx context.Context) (<-chan *logstore.LogEntry, error) {
	ctx, span := tracer.Start(ctx, "svc.Subscribe")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if !caller.IsAdminModeActivated(ctx) {
		return nil, errors.New("only admins with admin mode activated can subscribe to admin log tail events", errors.WithErrorCode(errors.EForbidden))
	}

	return s.store.Subscribe(ctx)
}
