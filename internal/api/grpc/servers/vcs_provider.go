// Package servers implements the gRPC servers.
package servers

import (
	"context"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// VCSProviderServer embeds the UnimplementedVCSProvidersServer.
type VCSProviderServer struct {
	pb.UnimplementedVCSProvidersServer
	serviceCatalog *services.Catalog
}

// NewVCSProviderServer returns an instance of VCSProviderServer.
func NewVCSProviderServer(serviceCatalog *services.Catalog) *VCSProviderServer {
	return &VCSProviderServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetVCSProviderByID returns a VCS Provider by an ID.
func (s *VCSProviderServer) GetVCSProviderByID(ctx context.Context, req *pb.GetVCSProviderByIDRequest) (*pb.VCSProvider, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	vcsProvider, ok := model.(*models.VCSProvider)
	if !ok {
		return nil, errors.New("VCS provider with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBVCSProvider(vcsProvider), nil
}

// GetVCSProviders returns a paginated list of VCS Providers.
func (s *VCSProviderServer) GetVCSProviders(ctx context.Context, req *pb.GetVCSProvidersRequest) (*pb.GetVCSProvidersResponse, error) {
	sort := db.VCSProviderSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &vcs.GetVCSProvidersInput{
		Search:            req.Search,
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		NamespacePath:     req.NamespacePath,
		IncludeInherited:  req.IncludeInherited,
	}

	result, err := s.serviceCatalog.VCSService.GetVCSProviders(ctx, input)
	if err != nil {
		return nil, err
	}

	vcsProviders := result.VCSProviders

	pbVCSProviders := make([]*pb.VCSProvider, len(vcsProviders))
	for ix := range vcsProviders {
		pbVCSProviders[ix] = toPBVCSProvider(&vcsProviders[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(vcsProviders) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&vcsProviders[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&vcsProviders[len(vcsProviders)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetVCSProvidersResponse{
		PageInfo:     pageInfo,
		VcsProviders: pbVCSProviders,
	}, nil
}

// CreateVCSProvider creates a new VCS Provider.
func (s *VCSProviderServer) CreateVCSProvider(ctx context.Context, req *pb.CreateVCSProviderRequest) (*pb.CreateVCSProviderResponse, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	input := &vcs.CreateVCSProviderInput{
		Name:               req.Name,
		Description:        req.Description,
		GroupID:            groupID,
		Type:               models.VCSProviderType(strings.ToLower(req.Type.String())),
		URL:                req.Url,
		OAuthClientID:      req.OauthClientId,
		OAuthClientSecret:  req.OauthClientSecret,
		AutoCreateWebhooks: req.AutoCreateWebhooks,
	}

	result, err := s.serviceCatalog.VCSService.CreateVCSProvider(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.CreateVCSProviderResponse{
		VcsProvider:           toPBVCSProvider(result.VCSProvider),
		OauthAuthorizationUrl: result.OAuthAuthorizationURL,
	}, nil
}

// UpdateVCSProvider returns the updated VCS Provider.
func (s *VCSProviderServer) UpdateVCSProvider(ctx context.Context, req *pb.UpdateVCSProviderRequest) (*pb.VCSProvider, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	provider, ok := model.(*models.VCSProvider)
	if !ok {
		return nil, errors.New("vcs provider with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		provider.Metadata.Version = int(*req.Version)
	}

	if req.Description != nil {
		provider.Description = *req.Description
	}

	if req.OauthClientId != nil {
		provider.OAuthClientID = *req.OauthClientId
	}

	if req.OauthClientSecret != nil {
		provider.OAuthClientSecret = *req.OauthClientSecret
	}

	input := &vcs.UpdateVCSProviderInput{
		Provider: provider,
	}

	updatedVCSProvider, err := s.serviceCatalog.VCSService.UpdateVCSProvider(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBVCSProvider(updatedVCSProvider), nil
}

// DeleteVCSProvider deletes a VCS Provider.
func (s *VCSProviderServer) DeleteVCSProvider(ctx context.Context, req *pb.DeleteVCSProviderRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	provider, ok := model.(*models.VCSProvider)
	if !ok {
		return nil, errors.New("vcs provider with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		provider.Metadata.Version = int(*req.Version)
	}

	input := &vcs.DeleteVCSProviderInput{
		Provider: provider,
		Force:    req.GetForce(),
	}

	if err := s.serviceCatalog.VCSService.DeleteVCSProvider(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ResetVCSProviderOAuthToken resets the OAuth token for a VCS Provider.
func (s *VCSProviderServer) ResetVCSProviderOAuthToken(ctx context.Context, req *pb.ResetVCSProviderOAuthTokenRequest) (*pb.ResetVCSProviderOAuthTokenResponse, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.ProviderId)
	if err != nil {
		return nil, err
	}

	gotVCSProvider, ok := model.(*models.VCSProvider)
	if !ok {
		return nil, errors.New("VCS provider with id %s not found", req.ProviderId, errors.WithErrorCode(errors.ENotFound))
	}

	input := &vcs.ResetVCSProviderOAuthTokenInput{
		VCSProvider: gotVCSProvider,
	}

	result, err := s.serviceCatalog.VCSService.ResetVCSProviderOAuthToken(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.ResetVCSProviderOAuthTokenResponse{
		VcsProvider:           toPBVCSProvider(result.VCSProvider),
		OauthAuthorizationUrl: result.OAuthAuthorizationURL,
	}, nil
}

// GetWorkspaceVCSProviderLinkByID returns a WorkspaceVCSProviderLink by an ID.
func (s *VCSProviderServer) GetWorkspaceVCSProviderLinkByID(ctx context.Context, req *pb.GetWorkspaceVCSProviderLinkByIDRequest) (*pb.WorkspaceVCSProviderLink, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	link, ok := model.(*models.WorkspaceVCSProviderLink)
	if !ok {
		return nil, errors.New("workspace VCS provider link with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBWorkspaceVCSProviderLink(link), nil
}

// CreateWorkspaceVCSProviderLink creates a new WorkspaceVCSProviderLink.
func (s *VCSProviderServer) CreateWorkspaceVCSProviderLink(ctx context.Context, req *pb.CreateWorkspaceVCSProviderLinkRequest) (*pb.CreateWorkspaceVCSProviderLinkResponse, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	workspace, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.WorkspaceId, errors.WithErrorCode(errors.ENotFound))
	}

	providerID, err := s.serviceCatalog.FetchModelID(ctx, req.ProviderId)
	if err != nil {
		return nil, err
	}

	input := &vcs.CreateWorkspaceVCSProviderLinkInput{
		Workspace:           workspace,
		ProviderID:          providerID,
		RepositoryPath:      req.RepositoryPath,
		ModuleDirectory:     req.ModuleDirectory,
		Branch:              req.Branch,
		TagRegex:            req.TagRegex,
		GlobPatterns:        req.GlobPatterns,
		AutoSpeculativePlan: req.AutoSpeculativePlan,
		WebhookDisabled:     req.WebhookDisabled,
	}

	result, err := s.serviceCatalog.VCSService.CreateWorkspaceVCSProviderLink(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.CreateWorkspaceVCSProviderLinkResponse{
		VcsProviderLink: toPBWorkspaceVCSProviderLink(result.Link),
		WebhookToken:    string(result.WebhookToken),
		WebhookUrl:      result.WebhookURL,
	}, nil
}

// UpdateWorkspaceVCSProviderLink updates a WorkspaceVCSProviderLink.
func (s *VCSProviderServer) UpdateWorkspaceVCSProviderLink(ctx context.Context, req *pb.UpdateWorkspaceVCSProviderLinkRequest) (*pb.WorkspaceVCSProviderLink, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	link, ok := model.(*models.WorkspaceVCSProviderLink)
	if !ok {
		return nil, errors.New("workspace vcs provider link with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		link.Metadata.Version = int(*req.Version)
	}

	if req.ModuleDirectory != nil {
		link.ModuleDirectory = req.ModuleDirectory
	}

	if req.Branch != nil {
		link.Branch = *req.Branch
	}

	if req.TagRegex != nil {
		link.TagRegex = req.TagRegex
	}

	if len(req.GlobPatterns) > 0 {
		link.GlobPatterns = req.GlobPatterns
	}

	if req.AutoSpeculativePlan != nil {
		link.AutoSpeculativePlan = *req.AutoSpeculativePlan
	}

	if req.WebhookDisabled != nil {
		link.WebhookDisabled = *req.WebhookDisabled
	}

	input := &vcs.UpdateWorkspaceVCSProviderLinkInput{
		Link: link,
	}

	updatedLink, err := s.serviceCatalog.VCSService.UpdateWorkspaceVCSProviderLink(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBWorkspaceVCSProviderLink(updatedLink), nil
}

// DeleteWorkspaceVCSProviderLink deletes a WorkspaceVCSProviderLink.
func (s *VCSProviderServer) DeleteWorkspaceVCSProviderLink(ctx context.Context, req *pb.DeleteWorkspaceVCSProviderLinkRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	link, ok := model.(*models.WorkspaceVCSProviderLink)
	if !ok {
		return nil, errors.New("workspace vcs provider link with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		link.Metadata.Version = int(*req.Version)
	}

	input := &vcs.DeleteWorkspaceVCSProviderLinkInput{
		Link:  link,
		Force: req.GetForce(),
	}

	if err := s.serviceCatalog.VCSService.DeleteWorkspaceVCSProviderLink(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateVCSRun creates a run from a VCS event.
func (s *VCSProviderServer) CreateVCSRun(ctx context.Context, req *pb.CreateVCSRunRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	workspace, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.WorkspaceId, errors.WithErrorCode(errors.ENotFound))
	}

	input := &vcs.CreateVCSRunInput{
		Workspace:     workspace,
		ReferenceName: req.ReferenceName,
		IsDestroy:     req.IsDestroy,
	}

	if err := s.serviceCatalog.VCSService.CreateVCSRun(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBVCSProvider converts from VCSProvider model to ProtoBuf model.
func toPBVCSProvider(v *models.VCSProvider) *pb.VCSProvider {
	return &pb.VCSProvider{
		Metadata:           toPBMetadata(&v.Metadata, types.VCSProviderModelType),
		Name:               v.Name,
		Description:        v.Description,
		GroupId:            gid.ToGlobalID(types.GroupModelType, v.GroupID),
		Type:               string(v.Type),
		Url:                v.URL.String(),
		CreatedBy:          v.CreatedBy,
		AutoCreateWebhooks: v.AutoCreateWebhooks,
	}
}

// toPBWorkspaceVCSProviderLink converts from WorkspaceVCSProviderLink model to ProtoBuf model.
func toPBWorkspaceVCSProviderLink(w *models.WorkspaceVCSProviderLink) *pb.WorkspaceVCSProviderLink {
	return &pb.WorkspaceVCSProviderLink{
		Metadata:            toPBMetadata(&w.Metadata, types.WorkspaceVCSProviderLinkModelType),
		CreatedBy:           w.CreatedBy,
		WorkspaceId:         gid.ToGlobalID(types.WorkspaceModelType, w.WorkspaceID),
		VcsProviderId:       gid.ToGlobalID(types.VCSProviderModelType, w.ProviderID),
		RepositoryPath:      w.RepositoryPath,
		WebhookId:           w.WebhookID,
		ModuleDirectory:     w.ModuleDirectory,
		Branch:              w.Branch,
		TagRegex:            w.TagRegex,
		GlobPatterns:        w.GlobPatterns,
		AutoSpeculativePlan: w.AutoSpeculativePlan,
		WebhookDisabled:     w.WebhookDisabled,
	}
}
