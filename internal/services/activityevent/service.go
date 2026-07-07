// Package activityevent package
package activityevent

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetActivityEventsInput is the input for querying a list of activity events
type GetActivityEventsInput struct {
	Sort              *db.ActivityEventSortableField
	PaginationOptions *pagination.Options
	UserID            *string
	ServiceAccountID  *string
	NamespacePath     *string
	TimeRangeStart    *time.Time
	TimeRangeEnd      *time.Time
	Actions           []models.ActivityEventAction
	TargetTypes       []models.ActivityEventTargetType
	IncludeNested     bool
}

// Service implements all activity event related functionality
type Service interface {
	GetActivityEvents(ctx context.Context, input *GetActivityEventsInput) (*db.ActivityEventsResult, error)
}

type service struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewService creates an instance of Service
func NewService(dbClient *db.Client, logger logger.Logger) Service {
	return &service{dbClient: dbClient, logger: logger}
}

func (s *service) GetActivityEvents(ctx context.Context,
	input *GetActivityEventsInput,
) (*db.ActivityEventsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetActivityEvents")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// A nil slice means no membership filter (admin sees all); a non-nil (possibly empty) slice
	// restricts results to the caller's root member namespaces and their descendants.
	var rootNamespaceMemberships []models.MembershipNamespace

	if !caller.IsAdminModeActivated(ctx) {
		rootNamespaces, rErr := caller.GetRootNamespaceMemberships(ctx)
		if rErr != nil {
			tracing.RecordError(span, rErr, "failed to get root namespaces")
			return nil, rErr
		}
		rootNamespaceMemberships = rootNamespaces
	}

	dbInput := db.GetActivityEventsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.ActivityEventFilter{
			UserID:           input.UserID,
			ServiceAccountID: input.ServiceAccountID,
			NamespacePath:    input.NamespacePath,
			IncludeNested:    input.IncludeNested,
			TimeRangeStart:   input.TimeRangeStart,
			TimeRangeEnd:     input.TimeRangeEnd,
			Actions:          input.Actions,
			TargetTypes:      input.TargetTypes,
			// RootNamespaceMemberships ensures the caller can only query events from namespaces
			// they are a member of (or descendants thereof).
			RootNamespaceMemberships: rootNamespaceMemberships,
		},
	}

	activityEventsResult, err := s.dbClient.ActivityEvents.GetActivityEvents(ctx, &dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get activity events")
		return nil, err
	}

	return activityEventsResult, nil
}
