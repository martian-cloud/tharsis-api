package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* ConfigurationVersion Query Resolvers */

// ConfigurationVersionQueryArgs are used to query a single configuration version
type ConfigurationVersionQueryArgs struct {
	ID string
}

// ConfigurationVersionResolver resolves a configuration version resource
type ConfigurationVersionResolver struct {
	configurationVersion *models.ConfigurationVersion
}

// ID resolver
func (r *ConfigurationVersionResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.ConfigurationVersionType, r.configurationVersion.Metadata.ID))
}

// Status resolver
func (r *ConfigurationVersionResolver) Status() string {
	return string(r.configurationVersion.Status)
}

// Speculative resolver
func (r *ConfigurationVersionResolver) Speculative() bool {
	return r.configurationVersion.Speculative
}

// WorkspaceID resolver
func (r *ConfigurationVersionResolver) WorkspaceID() string {
	return gid.ToGlobalID(gid.ConfigurationVersionType, r.configurationVersion.WorkspaceID)
}

// Metadata resolver
func (r *ConfigurationVersionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.configurationVersion.Metadata}
}

// CreatedBy resolver
func (r *ConfigurationVersionResolver) CreatedBy() string {
	return r.configurationVersion.CreatedBy
}

// VCSEvent resolver
func (r *ConfigurationVersionResolver) VCSEvent(ctx context.Context) (*VCSEventResolver, error) {
	if r.configurationVersion.VCSEventID == nil {
		return nil, nil
	}

	event, err := loadVCSEvent(ctx, *r.configurationVersion.VCSEventID)
	if err != nil {
		return nil, err
	}

	return &VCSEventResolver{vcsEvent: event}, nil
}

func configurationVersionQuery(ctx context.Context, args *ConfigurationVersionQueryArgs) (*ConfigurationVersionResolver, error) {
	service := getWorkspaceService(ctx)

	cv, err := service.GetConfigurationVersion(ctx, gid.FromGlobalID(args.ID))
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	if cv == nil {
		return nil, nil
	}

	return &ConfigurationVersionResolver{configurationVersion: cv}, nil
}

/* ConfigurationVersion Mutations */

// ConfigurationVersionMutationPayload is the response payload for a configuration version mutation
type ConfigurationVersionMutationPayload struct {
	ClientMutationID     *string
	ConfigurationVersion *models.ConfigurationVersion
	Problems             []Problem
}

// ConfigurationVersionMutationPayloadResolver resolves a ConfigurationVersionMutationPayload
type ConfigurationVersionMutationPayloadResolver struct {
	ConfigurationVersionMutationPayload
}

// ConfigurationVersion field resolver
func (r *ConfigurationVersionMutationPayloadResolver) ConfigurationVersion() *ConfigurationVersionResolver {
	if r.ConfigurationVersionMutationPayload.ConfigurationVersion == nil {
		return nil
	}
	return &ConfigurationVersionResolver{configurationVersion: r.ConfigurationVersionMutationPayload.ConfigurationVersion}
}

// CreateConfigurationVersionInput is the input for creating a new configuration version
type CreateConfigurationVersionInput struct {
	ClientMutationID *string
	Speculative      *bool
	WorkspacePath    string
}

func handleConfigurationVersionMutationProblem(e error, clientMutationID *string) (*ConfigurationVersionMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := ConfigurationVersionMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ConfigurationVersionMutationPayloadResolver{ConfigurationVersionMutationPayload: payload}, nil
}

func createConfigurationVersionMutation(ctx context.Context, input *CreateConfigurationVersionInput) (*ConfigurationVersionMutationPayloadResolver, error) {
	ws, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, input.WorkspacePath)
	if err != nil {
		return nil, err
	}

	options := &workspace.CreateConfigurationVersionInput{
		WorkspaceID: ws.Metadata.ID,
	}

	if input.Speculative != nil {
		options.Speculative = *input.Speculative
	}

	cv, err := getWorkspaceService(ctx).CreateConfigurationVersion(ctx, options)
	if err != nil {
		return nil, err
	}

	payload := ConfigurationVersionMutationPayload{ClientMutationID: input.ClientMutationID, ConfigurationVersion: cv, Problems: []Problem{}}
	return &ConfigurationVersionMutationPayloadResolver{ConfigurationVersionMutationPayload: payload}, nil
}

/* ConfigurationVersion loader */

const configurationVersionLoaderKey = "configurationVersion"

// RegisterConfigurationVersionLoader registers a configurationVersion loader function
func RegisterConfigurationVersionLoader(collection *loader.Collection) {
	collection.Register(configurationVersionLoaderKey, configurationVersionBatchFunc)
}

func loadConfigurationVersion(ctx context.Context, id string) (*models.ConfigurationVersion, error) {
	ldr, err := loader.Extract(ctx, configurationVersionLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	configurationVersion, ok := data.(models.ConfigurationVersion)
	if !ok {
		return nil, errors.New(errors.EInternal, "Wrong type")
	}

	return &configurationVersion, nil
}

func configurationVersionBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	configurationVersions, err := getWorkspaceService(ctx).GetConfigurationVersionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range configurationVersions {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
