package resolver

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// MaintenanceModeResolver resolves a MaintenanceMode
type MaintenanceModeResolver struct {
	maintenanceMode *models.MaintenanceMode
}

// ID resolver
func (r *MaintenanceModeResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.MaintenanceModeType, r.maintenanceMode.Metadata.ID))
}

// Metadata resolver
func (r *MaintenanceModeResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.maintenanceMode.Metadata}
}

// CreatedBy resolver
func (r *MaintenanceModeResolver) CreatedBy() string {
	return r.maintenanceMode.CreatedBy
}

// Message resolver
func (r *MaintenanceModeResolver) Message() string {
	return r.maintenanceMode.Message
}

func maintenanceModeQuery(ctx context.Context) (*MaintenanceModeResolver, error) {
	maintenanceMode, err := getMaintenanceModeService(ctx).GetMaintenanceMode(ctx)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}

		return nil, err
	}

	return &MaintenanceModeResolver{maintenanceMode: maintenanceMode}, nil
}

// MaintenanceModeMutationPayload is the response payload for a maintenance mode mutation.
type MaintenanceModeMutationPayload struct {
	ClientMutationID *string
	MaintenanceMode  *models.MaintenanceMode
	Problems         []Problem
}

// MaintenanceModeMutationPayloadResolver resolves a MaintenanceModeMutationPayload
type MaintenanceModeMutationPayloadResolver struct {
	MaintenanceModeMutationPayload
}

// MaintenanceMode resolver
func (r *MaintenanceModeMutationPayloadResolver) MaintenanceMode() *MaintenanceModeResolver {
	if r.MaintenanceModeMutationPayload.MaintenanceMode == nil {
		return nil
	}

	return &MaintenanceModeResolver{r.MaintenanceModeMutationPayload.MaintenanceMode}
}

// EnableMaintenanceModeInput is the input for enabling maintenance mode.
type EnableMaintenanceModeInput struct {
	ClientMutationID *string
	Message          string
}

// DisableMaintenanceModeInput is the input for disabling maintenance mode.
type DisableMaintenanceModeInput struct {
	ClientMutationID *string
}

func handleMaintenanceModeMutationProblem(e error, clientMutationID *string) (*MaintenanceModeMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := MaintenanceModeMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &MaintenanceModeMutationPayloadResolver{MaintenanceModeMutationPayload: payload}, nil
}

func enableMaintenanceModeMutation(ctx context.Context, input *EnableMaintenanceModeInput) (*MaintenanceModeMutationPayloadResolver, error) {
	toCreate := &maintenance.EnableMaintenanceModeInput{
		Message: input.Message,
	}

	maintenanceMode, err := getMaintenanceModeService(ctx).EnableMaintenanceMode(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	payload := MaintenanceModeMutationPayload{ClientMutationID: input.ClientMutationID, MaintenanceMode: maintenanceMode, Problems: []Problem{}}
	return &MaintenanceModeMutationPayloadResolver{MaintenanceModeMutationPayload: payload}, nil
}

func disableMaintenanceModeMutation(ctx context.Context, input *DisableMaintenanceModeInput) (*MaintenanceModeMutationPayloadResolver, error) {
	service := getMaintenanceModeService(ctx)

	maintenanceMode, err := service.GetMaintenanceMode(ctx)
	if err != nil {
		return nil, err
	}

	if err = service.DisableMaintenanceMode(ctx); err != nil {
		return nil, err
	}

	payload := MaintenanceModeMutationPayload{ClientMutationID: input.ClientMutationID, MaintenanceMode: maintenanceMode, Problems: []Problem{}}
	return &MaintenanceModeMutationPayloadResolver{MaintenanceModeMutationPayload: payload}, nil
}
