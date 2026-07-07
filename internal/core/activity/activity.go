// Package activity provides the internal audit-log writer for activity events.
package activity

import (
	"context"
	"encoding/json"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
)

// CreateActivityEventInput specifies the inputs for creating an activity event.
// The function will assign the user or service account caller.
type CreateActivityEventInput struct {
	NamespacePath *string
	Payload       any
	Action        models.ActivityEventAction
	TargetType    models.ActivityEventTargetType
	TargetID      string
}

// CreateActivityEvent records an activity event attributed to the caller on the
// context. It is an internal audit-log writer: if the caller is neither a user
// nor a service account, no event is recorded and (nil, nil) is returned.
func CreateActivityEvent(ctx context.Context, dbClient *db.Client, input *CreateActivityEventInput) (*models.ActivityEvent, error) {
	ctx, span := tracer.Start(ctx, "activity.CreateActivityEvent")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
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

	activityEvent, err := dbClient.ActivityEvents.CreateActivityEvent(ctx, &toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	return activityEvent, nil
}
