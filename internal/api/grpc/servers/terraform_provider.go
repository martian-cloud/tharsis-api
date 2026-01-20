// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TerraformProviderServer embeds the UnimplementedTerraformProvidersServer.
type TerraformProviderServer struct {
	pb.UnimplementedTerraformProvidersServer
	serviceCatalog *services.Catalog
}

// NewTerraformProviderServer returns an instance of TerraformProviderServer.
func NewTerraformProviderServer(serviceCatalog *services.Catalog) *TerraformProviderServer {
	return &TerraformProviderServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetTerraformProviderByID returns a TerraformProvider by an ID.
func (s *TerraformProviderServer) GetTerraformProviderByID(ctx context.Context, req *pb.GetTerraformProviderByIDRequest) (*pb.TerraformProvider, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	provider, ok := model.(*models.TerraformProvider)
	if !ok {
		return nil, errors.New("terraform provider with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformProvider(provider), nil
}

// GetTerraformProviders returns a paginated list of TerraformProviders.
func (s *TerraformProviderServer) GetTerraformProviders(ctx context.Context, req *pb.GetTerraformProvidersRequest) (*pb.GetTerraformProvidersResponse, error) {
	sort := db.TerraformProviderSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	var group *models.Group
	if req.GroupId != nil {
		model, gErr := s.serviceCatalog.FetchModel(ctx, *req.GroupId)
		if gErr != nil {
			return nil, gErr
		}

		var ok bool
		group, ok = model.(*models.Group)
		if !ok {
			return nil, errors.New("group with id %s not found", *req.GroupId, errors.WithErrorCode(errors.ENotFound))
		}
	}

	input := &providerregistry.GetProvidersInput{
		Search:            req.Search,
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		Group:             group,
	}

	result, err := s.serviceCatalog.TerraformProviderRegistryService.GetProviders(ctx, input)
	if err != nil {
		return nil, err
	}

	providers := result.Providers

	pbProviders := make([]*pb.TerraformProvider, len(providers))
	for ix := range providers {
		pbProviders[ix] = toPBTerraformProvider(&providers[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(providers) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&providers[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&providers[len(providers)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTerraformProvidersResponse{
		PageInfo:  pageInfo,
		Providers: pbProviders,
	}, nil
}

// CreateTerraformProvider creates a new TerraformProvider.
func (s *TerraformProviderServer) CreateTerraformProvider(ctx context.Context, req *pb.CreateTerraformProviderRequest) (*pb.TerraformProvider, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	input := &providerregistry.CreateProviderInput{
		Name:          req.Name,
		GroupID:       groupID,
		RepositoryURL: req.RepositoryUrl,
		Private:       req.Private,
	}

	createdProvider, err := s.serviceCatalog.TerraformProviderRegistryService.CreateProvider(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTerraformProvider(createdProvider), nil
}

// UpdateTerraformProvider returns the updated TerraformProvider.
func (s *TerraformProviderServer) UpdateTerraformProvider(ctx context.Context, req *pb.UpdateTerraformProviderRequest) (*pb.TerraformProvider, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotProvider, ok := model.(*models.TerraformProvider)
	if !ok {
		return nil, errors.New("terraform provider with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotProvider.Metadata.Version = int(*req.Version)
	}

	if req.RepositoryUrl != nil {
		gotProvider.RepositoryURL = *req.RepositoryUrl
	}

	if req.Private != nil {
		gotProvider.Private = *req.Private
	}

	updatedProvider, err := s.serviceCatalog.TerraformProviderRegistryService.UpdateProvider(ctx, gotProvider)
	if err != nil {
		return nil, err
	}

	return toPBTerraformProvider(updatedProvider), nil
}

// DeleteTerraformProvider deletes a TerraformProvider.
func (s *TerraformProviderServer) DeleteTerraformProvider(ctx context.Context, req *pb.DeleteTerraformProviderRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotProvider, ok := model.(*models.TerraformProvider)
	if !ok {
		return nil, errors.New("terraform provider with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformProviderRegistryService.DeleteProvider(ctx, gotProvider); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetTerraformProviderVersionByID returns a TerraformProviderVersion by ID.
func (s *TerraformProviderServer) GetTerraformProviderVersionByID(ctx context.Context, req *pb.GetTerraformProviderVersionByIDRequest) (*pb.TerraformProviderVersion, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	version, ok := model.(*models.TerraformProviderVersion)
	if !ok {
		return nil, errors.New("terraform provider version with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformProviderVersion(version), nil
}

// GetTerraformProviderVersions returns a paginated list of TerraformProviderVersions.
func (s *TerraformProviderServer) GetTerraformProviderVersions(ctx context.Context, req *pb.GetTerraformProviderVersionsRequest) (*pb.GetTerraformProviderVersionsResponse, error) {
	sort := db.TerraformProviderVersionSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	providerID, err := s.serviceCatalog.FetchModelID(ctx, req.ProviderId)
	if err != nil {
		return nil, err
	}

	input := &providerregistry.GetProviderVersionsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		ProviderID:        providerID,
		SemanticVersion:   req.SemanticVersion,
		Latest:            req.Latest,
	}

	result, err := s.serviceCatalog.TerraformProviderRegistryService.GetProviderVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	versions := result.ProviderVersions

	pbVersions := make([]*pb.TerraformProviderVersion, len(versions))
	for ix := range versions {
		pbVersions[ix] = toPBTerraformProviderVersion(&versions[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(versions) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&versions[0])
		if err != nil {
			return nil, err
		}
		pageInfo.EndCursor, err = result.PageInfo.Cursor(&versions[len(versions)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTerraformProviderVersionsResponse{
		PageInfo: pageInfo,
		Versions: pbVersions,
	}, nil
}

// CreateTerraformProviderVersion creates a new TerraformProviderVersion.
func (s *TerraformProviderServer) CreateTerraformProviderVersion(ctx context.Context, req *pb.CreateTerraformProviderVersionRequest) (*pb.TerraformProviderVersion, error) {
	providerID, err := s.serviceCatalog.FetchModelID(ctx, req.ProviderId)
	if err != nil {
		return nil, err
	}

	input := &providerregistry.CreateProviderVersionInput{
		ProviderID:      providerID,
		SemanticVersion: req.Version,
		Protocols:       req.Protocols,
	}

	created, err := s.serviceCatalog.TerraformProviderRegistryService.CreateProviderVersion(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTerraformProviderVersion(created), nil
}

// DeleteTerraformProviderVersion deletes a TerraformProviderVersion.
func (s *TerraformProviderServer) DeleteTerraformProviderVersion(ctx context.Context, req *pb.DeleteTerraformProviderVersionRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	version, ok := model.(*models.TerraformProviderVersion)
	if !ok {
		return nil, errors.New("terraform provider version with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformProviderRegistryService.DeleteProviderVersion(ctx, version); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetTerraformProviderPlatformByID returns a TerraformProviderPlatform by ID.
func (s *TerraformProviderServer) GetTerraformProviderPlatformByID(ctx context.Context, req *pb.GetTerraformProviderPlatformByIDRequest) (*pb.TerraformProviderPlatform, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	platform, ok := model.(*models.TerraformProviderPlatform)
	if !ok {
		return nil, errors.New("terraform provider platform with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformProviderPlatform(platform), nil
}

// GetTerraformProviderPlatforms returns a paginated list of TerraformProviderPlatforms.
func (s *TerraformProviderServer) GetTerraformProviderPlatforms(ctx context.Context, req *pb.GetTerraformProviderPlatformsRequest) (*pb.GetTerraformProviderPlatformsResponse, error) {
	sort := db.TerraformProviderPlatformSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &providerregistry.GetProviderPlatformsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		BinaryUploaded:    req.BinaryUploaded,
	}

	if req.ProviderVersionId != nil {
		versionID, vErr := s.serviceCatalog.FetchModelID(ctx, *req.ProviderVersionId)
		if vErr != nil {
			return nil, vErr
		}
		input.ProviderVersionID = &versionID
	}

	if req.ProviderId != nil {
		providerID, pErr := s.serviceCatalog.FetchModelID(ctx, *req.ProviderId)
		if pErr != nil {
			return nil, pErr
		}
		input.ProviderID = &providerID
	}

	result, err := s.serviceCatalog.TerraformProviderRegistryService.GetProviderPlatforms(ctx, input)
	if err != nil {
		return nil, err
	}

	platforms := result.ProviderPlatforms

	pbPlatforms := make([]*pb.TerraformProviderPlatform, len(platforms))
	for ix := range platforms {
		pbPlatforms[ix] = toPBTerraformProviderPlatform(&platforms[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(platforms) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&platforms[0])
		if err != nil {
			return nil, err
		}
		pageInfo.EndCursor, err = result.PageInfo.Cursor(&platforms[len(platforms)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTerraformProviderPlatformsResponse{
		PageInfo:  pageInfo,
		Platforms: pbPlatforms,
	}, nil
}

// CreateTerraformProviderPlatform creates a new TerraformProviderPlatform.
func (s *TerraformProviderServer) CreateTerraformProviderPlatform(ctx context.Context, req *pb.CreateTerraformProviderPlatformRequest) (*pb.TerraformProviderPlatform, error) {
	versionID, err := s.serviceCatalog.FetchModelID(ctx, req.ProviderVersionId)
	if err != nil {
		return nil, err
	}

	input := &providerregistry.CreateProviderPlatformInput{
		ProviderVersionID: versionID,
		OperatingSystem:   req.Os,
		Architecture:      req.Arch,
		SHASum:            req.ShaSum,
		Filename:          req.Filename,
	}

	created, err := s.serviceCatalog.TerraformProviderRegistryService.CreateProviderPlatform(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTerraformProviderPlatform(created), nil
}

// DeleteTerraformProviderPlatform deletes a TerraformProviderPlatform.
func (s *TerraformProviderServer) DeleteTerraformProviderPlatform(ctx context.Context, req *pb.DeleteTerraformProviderPlatformRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	platform, ok := model.(*models.TerraformProviderPlatform)
	if !ok {
		return nil, errors.New("terraform provider platform with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformProviderRegistryService.DeleteProviderPlatform(ctx, platform); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBTerraformProvider converts from TerraformProvider model to ProtoBuf model.
func toPBTerraformProvider(p *models.TerraformProvider) *pb.TerraformProvider {
	return &pb.TerraformProvider{
		Metadata:      toPBMetadata(&p.Metadata, types.TerraformProviderModelType),
		CreatedBy:     p.CreatedBy,
		GroupId:       gid.ToGlobalID(types.GroupModelType, p.GroupID),
		Name:          p.Name,
		Private:       p.Private,
		RepositoryUrl: p.RepositoryURL,
	}
}

func toPBTerraformProviderVersion(v *models.TerraformProviderVersion) *pb.TerraformProviderVersion {
	var gpgKeyID *uint64
	if v.GPGKeyID != nil {
		gpgKeyID = v.GPGKeyID
	}

	return &pb.TerraformProviderVersion{
		Metadata:           toPBMetadata(&v.Metadata, types.TerraformProviderVersionModelType),
		CreatedBy:          v.CreatedBy,
		GpgAsciiArmor:      v.GPGASCIIArmor,
		GpgKeyId:           gpgKeyID,
		Latest:             v.Latest,
		Protocols:          v.Protocols,
		ProviderId:         gid.ToGlobalID(types.TerraformProviderModelType, v.ProviderID),
		ReadmeUploaded:     v.ReadmeUploaded,
		SemanticVersion:    v.SemanticVersion,
		ShaSumsSigUploaded: v.SHASumsSignatureUploaded,
		ShaSumsUploaded:    v.SHASumsUploaded,
	}
}

func toPBTerraformProviderPlatform(p *models.TerraformProviderPlatform) *pb.TerraformProviderPlatform {
	return &pb.TerraformProviderPlatform{
		Metadata:          toPBMetadata(&p.Metadata, types.TerraformProviderPlatformModelType),
		Architecture:      p.Architecture,
		BinaryUploaded:    p.BinaryUploaded,
		CreatedBy:         p.CreatedBy,
		Filename:          p.Filename,
		OperatingSystem:   p.OperatingSystem,
		ProviderVersionId: gid.ToGlobalID(types.TerraformProviderVersionModelType, p.ProviderVersionID),
		ShaSum:            p.SHASum,
	}
}
