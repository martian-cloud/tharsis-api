package resolver

import (
	"context"
	"path/filepath"
	"strconv"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
)

/* VCSProviderLink Query Resolvers */

// WorkspaceVCSProviderLinkResolver resolves a workspaceVCSProviderLink resource
type WorkspaceVCSProviderLinkResolver struct {
	workspaceVCSProviderLink *models.WorkspaceVCSProviderLink
}

// ID resolver
func (r *WorkspaceVCSProviderLinkResolver) ID() graphql.ID {
	return graphql.ID(r.workspaceVCSProviderLink.GetGlobalID())
}

// CreatedBy resolver
func (r *WorkspaceVCSProviderLinkResolver) CreatedBy() string {
	return r.workspaceVCSProviderLink.CreatedBy
}

// Workspace resolver
func (r *WorkspaceVCSProviderLinkResolver) Workspace(ctx context.Context) (*WorkspaceResolver, error) {
	workspace, err := loadWorkspace(ctx, r.workspaceVCSProviderLink.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return &WorkspaceResolver{workspace: workspace}, nil
}

// VCSProvider resolver
func (r *WorkspaceVCSProviderLinkResolver) VCSProvider(ctx context.Context) (*VCSProviderResolver, error) {
	vcsProvider, err := loadVCSProvider(ctx, r.workspaceVCSProviderLink.ProviderID)
	if err != nil {
		return nil, err
	}

	return &VCSProviderResolver{vcsProvider: vcsProvider}, nil
}

// RepositoryPath resolver
func (r *WorkspaceVCSProviderLinkResolver) RepositoryPath() string {
	return r.workspaceVCSProviderLink.RepositoryPath
}

// WebhookID resolver
func (r *WorkspaceVCSProviderLinkResolver) WebhookID() *string {
	if r.workspaceVCSProviderLink.WebhookID == "" {
		return nil
	}

	return &r.workspaceVCSProviderLink.WebhookID
}

// ModuleDirectory resolver
func (r *WorkspaceVCSProviderLinkResolver) ModuleDirectory() *string {
	return r.workspaceVCSProviderLink.ModuleDirectory
}

// Branch resolver
func (r *WorkspaceVCSProviderLinkResolver) Branch() string {
	return r.workspaceVCSProviderLink.Branch
}

// TagRegex resolver
func (r *WorkspaceVCSProviderLinkResolver) TagRegex() *string {
	return r.workspaceVCSProviderLink.TagRegex
}

// GlobPatterns resolver
func (r *WorkspaceVCSProviderLinkResolver) GlobPatterns() []string {
	return r.workspaceVCSProviderLink.GlobPatterns
}

// Metadata resolver
func (r *WorkspaceVCSProviderLinkResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.workspaceVCSProviderLink.Metadata}
}

// AutoSpeculativePlan resolver
func (r *WorkspaceVCSProviderLinkResolver) AutoSpeculativePlan() bool {
	return r.workspaceVCSProviderLink.AutoSpeculativePlan
}

// WebhookDisabled resolver
func (r *WorkspaceVCSProviderLinkResolver) WebhookDisabled() bool {
	return r.workspaceVCSProviderLink.WebhookDisabled
}

/* WorkspaceVCSProviderLink Mutation Resolvers */

// WorkspaceVCSProviderLinkMutationPayload is the response payload for a workspace vcs provider mutation
type WorkspaceVCSProviderLinkMutationPayload struct {
	ClientMutationID *string
	VCSProviderLink  *models.WorkspaceVCSProviderLink
	Problems         []Problem
}

// WorkspaceVCSProviderLinkMutationPayloadResolver resolver a WorkspaceVCSProviderLinkMutationPayload
type WorkspaceVCSProviderLinkMutationPayloadResolver struct {
	webhookToken []byte
	webhookURL   *string
	WorkspaceVCSProviderLinkMutationPayload
}

// VCSProviderLink field resolver
func (r *WorkspaceVCSProviderLinkMutationPayloadResolver) VCSProviderLink() *WorkspaceVCSProviderLinkResolver {
	if r.WorkspaceVCSProviderLinkMutationPayload.VCSProviderLink == nil {
		return nil
	}

	return &WorkspaceVCSProviderLinkResolver{workspaceVCSProviderLink: r.WorkspaceVCSProviderLinkMutationPayload.VCSProviderLink}
}

// WebhookToken field resolver
func (r *WorkspaceVCSProviderLinkMutationPayloadResolver) WebhookToken() *string {
	if r.webhookToken == nil {
		return nil
	}

	tokenString := string(r.webhookToken)
	return &tokenString
}

// WebhookURL field resolver
func (r *WorkspaceVCSProviderLinkMutationPayloadResolver) WebhookURL() *string {
	return r.webhookURL
}

// CreateWorkspaceVCSProviderLinkInput is the input for creating a workspace VCS provider link.
type CreateWorkspaceVCSProviderLinkInput struct {
	ClientMutationID    *string
	ModuleDirectory     *string
	Branch              *string
	TagRegex            *string
	WorkspacePath       *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID         *string
	ProviderID          string
	RepositoryPath      string
	GlobPatterns        []string
	AutoSpeculativePlan bool
	WebhookDisabled     bool
}

// UpdateWorkspaceVCSProviderLinkInput is the input for updating a workspace VCS provider link.
type UpdateWorkspaceVCSProviderLinkInput struct {
	ClientMutationID    *string
	Metadata            *MetadataInput
	ModuleDirectory     *string
	TagRegex            *string
	Branch              *string
	AutoSpeculativePlan *bool
	WebhookDisabled     *bool
	ID                  string
	GlobPatterns        []string
}

// DeleteWorkspaceVCSProviderLinkInput is the input for deleting a workspace VCS provider link.
type DeleteWorkspaceVCSProviderLinkInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Force            *bool
	ID               string
}

func handleWorkspaceVCSProviderLinkMutationProblem(e error, clientMutationID *string) (*WorkspaceVCSProviderLinkMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := WorkspaceVCSProviderLinkMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &WorkspaceVCSProviderLinkMutationPayloadResolver{WorkspaceVCSProviderLinkMutationPayload: payload}, nil
}

func createWorkspaceVCSProviderLinkMutation(ctx context.Context, input *CreateWorkspaceVCSProviderLinkInput) (*WorkspaceVCSProviderLinkMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	serviceCatalog := getServiceCatalog(ctx)

	workspace, err := serviceCatalog.WorkspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	linkCreateOptions := &vcs.CreateWorkspaceVCSProviderLinkInput{
		Workspace:           workspace,
		ProviderID:          gid.FromGlobalID(input.ProviderID),
		RepositoryPath:      input.RepositoryPath,
		GlobPatterns:        input.GlobPatterns,
		AutoSpeculativePlan: input.AutoSpeculativePlan,
		ModuleDirectory:     input.ModuleDirectory,
		Branch:              input.Branch,
		TagRegex:            input.TagRegex,
		WebhookDisabled:     input.WebhookDisabled,
	}

	response, err := serviceCatalog.VCSService.CreateWorkspaceVCSProviderLink(ctx, linkCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := WorkspaceVCSProviderLinkMutationPayload{
		ClientMutationID: input.ClientMutationID,
		VCSProviderLink:  response.Link,
		Problems:         []Problem{},
	}

	return &WorkspaceVCSProviderLinkMutationPayloadResolver{
		WorkspaceVCSProviderLinkMutationPayload: payload,
		webhookToken:                            response.WebhookToken,
		webhookURL:                              response.WebhookURL,
	}, nil
}

func updateWorkspaceVCSProviderLinkMutation(ctx context.Context, input *UpdateWorkspaceVCSProviderLinkInput) (*WorkspaceVCSProviderLinkMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	linkID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	link, err := serviceCatalog.VCSService.GetWorkspaceVCSProviderLinkByID(ctx, linkID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		link.Metadata.Version = v
	}

	// Update fields
	link.GlobPatterns = input.GlobPatterns

	// Allow setting moduleDirectory to an empty string.
	// filepath.Clean() will return '.' for empty string.
	if input.ModuleDirectory != nil {
		if *input.ModuleDirectory != "" {
			moduleDir := filepath.Clean(*input.ModuleDirectory)
			link.ModuleDirectory = &moduleDir
		} else {
			link.ModuleDirectory = input.ModuleDirectory
		}
	}

	if input.TagRegex != nil {
		link.TagRegex = input.TagRegex
	}

	if input.Branch != nil && *input.Branch != "" {
		link.Branch = *input.Branch
	}

	if input.AutoSpeculativePlan != nil {
		link.AutoSpeculativePlan = *input.AutoSpeculativePlan
	}

	if input.WebhookDisabled != nil {
		link.WebhookDisabled = *input.WebhookDisabled
	}

	updatedLink, err := serviceCatalog.VCSService.UpdateWorkspaceVCSProviderLink(ctx, &vcs.UpdateWorkspaceVCSProviderLinkInput{Link: link})
	if err != nil {
		return nil, err
	}

	payload := WorkspaceVCSProviderLinkMutationPayload{ClientMutationID: input.ClientMutationID, VCSProviderLink: updatedLink, Problems: []Problem{}}
	return &WorkspaceVCSProviderLinkMutationPayloadResolver{WorkspaceVCSProviderLinkMutationPayload: payload}, nil
}

func deleteWorkspaceVCSProviderLinkMutation(ctx context.Context, input *DeleteWorkspaceVCSProviderLinkInput) (*WorkspaceVCSProviderLinkMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	linkID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	link, err := serviceCatalog.VCSService.GetWorkspaceVCSProviderLinkByID(ctx, linkID)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, sErr := strconv.Atoi(input.Metadata.Version)
		if sErr != nil {
			return nil, sErr
		}

		link.Metadata.Version = v
	}

	toDelete := &vcs.DeleteWorkspaceVCSProviderLinkInput{
		Link: link,
	}

	if input.Force != nil {
		toDelete.Force = *input.Force
	}

	if err = serviceCatalog.VCSService.DeleteWorkspaceVCSProviderLink(ctx, toDelete); err != nil {
		return nil, err
	}

	payload := WorkspaceVCSProviderLinkMutationPayload{ClientMutationID: input.ClientMutationID, VCSProviderLink: link, Problems: []Problem{}}
	return &WorkspaceVCSProviderLinkMutationPayloadResolver{WorkspaceVCSProviderLinkMutationPayload: payload}, nil
}

/* CreateVCSRun Mutation Resolvers */

// CreateVCSRunMutationPayload is the response payload for creating a vcs run.
type CreateVCSRunMutationPayload struct {
	ClientMutationID *string
	Problems         []Problem
}

// CreateVCSRunInput is the input for creating a VCS run.
type CreateVCSRunInput struct {
	ClientMutationID *string
	ReferenceName    *string
	IsDestroy        *bool
	WorkspacePath    *string // DEPRECATED: use WorkspaceID instead with a TRN
	WorkspaceID      *string
}

func handleVCSRunMutationProblem(e error, clientMutationID *string) (*CreateVCSRunMutationPayload, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	return &CreateVCSRunMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}, nil
}

func createVCSRunMutation(ctx context.Context, input *CreateVCSRunInput) (*CreateVCSRunMutationPayload, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	serviceCatalog := getServiceCatalog(ctx)

	ws, err := serviceCatalog.WorkspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	runOptions := &vcs.CreateVCSRunInput{
		Workspace:     ws,
		ReferenceName: input.ReferenceName,
	}

	if input.IsDestroy != nil {
		runOptions.IsDestroy = *input.IsDestroy
	}

	if err = serviceCatalog.VCSService.CreateVCSRun(ctx, runOptions); err != nil {
		return nil, err
	}

	return &CreateVCSRunMutationPayload{ClientMutationID: input.ClientMutationID, Problems: []Problem{}}, nil
}
