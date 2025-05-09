// Package activityevent package
package activityevent

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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

// CreateActivityEventInput specifies the inputs for creating an activity event
// The method will assign the user or service account caller.
type CreateActivityEventInput struct {
	NamespacePath *string
	Payload       interface{}
	Action        models.ActivityEventAction
	TargetType    models.ActivityEventTargetType
	TargetID      string
}

// Service implements all activity event related functionality
type Service interface {
	GetActivityEvents(ctx context.Context, input *GetActivityEventsInput) (*db.ActivityEventsResult, error)
	CreateActivityEvent(ctx context.Context, input *CreateActivityEventInput) (*models.ActivityEvent, error)
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
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	accessPolicy, err := caller.GetNamespaceAccessPolicy(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace access policy")
		return nil, err
	}

	var membershipRequirement *db.ActivityEventNamespaceMembershipRequirement

	if !accessPolicy.AllowAll {
		switch c := caller.(type) {
		case *auth.UserCaller:
			membershipRequirement = &db.ActivityEventNamespaceMembershipRequirement{UserID: &c.User.Metadata.ID}
		case *auth.ServiceAccountCaller:
			membershipRequirement = &db.ActivityEventNamespaceMembershipRequirement{ServiceAccountID: &c.ServiceAccountID}
		default:
			tracing.RecordError(span, nil, "invalid caller type: %T", caller)
			return nil, errors.New("invalid caller type", errors.WithErrorCode(errors.EUnauthorized))
		}
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
			// The NamespaceMembershipRequirement filter will verify that the caller can only query events
			// from namespaces they are a member of
			NamespaceMembershipRequirement: membershipRequirement,
		},
	}

	activityEventsResult, err := s.dbClient.ActivityEvents.GetActivityEvents(ctx, &dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get activity events")
		return nil, err
	}

	return activityEventsResult, nil
}

func (s *service) CreateActivityEvent(ctx context.Context, input *CreateActivityEventInput) (*models.ActivityEvent, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateActivityEvent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to authorize caller")
		return nil, err
	}

	var userID, serviceAccountID *string
	switch c := caller.(type) {
	case *auth.UserCaller:
		userID = &c.User.Metadata.ID
	case *auth.ServiceAccountCaller:
		serviceAccountID = &c.ServiceAccountID
	default:
		// If caller is not a user or service account, do nothing.
		return nil, nil
	}

	var payloadBuffer []byte
	if input.Payload != nil {
		payloadBuffer, err = json.Marshal(input.Payload)
		if err != nil {
			tracing.RecordError(span, err, "failed to marshal payload")
			return nil, err
		}
	}

	toCreate := models.ActivityEvent{
		UserID:           userID,
		ServiceAccountID: serviceAccountID,
		NamespacePath:    input.NamespacePath,
		Action:           input.Action,
		TargetType:       input.TargetType,
		TargetID:         input.TargetID,
		Payload:          payloadBuffer,
	}

	activityEvent, err := s.dbClient.ActivityEvents.CreateActivityEvent(ctx, &toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	return activityEvent, nil
}
