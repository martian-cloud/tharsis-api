// Package servers implements the gRPC servers.
package servers

import (
	"context"
	"encoding/hex"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TerraformModuleServer embeds the UnimplementedTerraformModulesServer.
type TerraformModuleServer struct {
	pb.UnimplementedTerraformModulesServer
	serviceCatalog *services.Catalog
}

// NewTerraformModuleServer returns an instance of TerraformModuleServer.
func NewTerraformModuleServer(serviceCatalog *services.Catalog) *TerraformModuleServer {
	return &TerraformModuleServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetTerraformModuleByID returns a TerraformModule by an ID.
func (s *TerraformModuleServer) GetTerraformModuleByID(ctx context.Context, req *pb.GetTerraformModuleByIDRequest) (*pb.TerraformModule, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	terraformModule, ok := model.(*models.TerraformModule)
	if !ok {
		return nil, errors.New("terraform module with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformModule(terraformModule), nil
}

// GetTerraformModules returns a paginated list of TerraformModules.
func (s *TerraformModuleServer) GetTerraformModules(ctx context.Context, req *pb.GetTerraformModulesRequest) (*pb.GetTerraformModulesResponse, error) {
	sort := db.TerraformModuleSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &moduleregistry.GetModulesInput{
		Search:            req.Search,
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		IncludeInherited:  req.IncludeInherited,
	}

	if req.GroupId != nil {
		model, err := s.serviceCatalog.FetchModel(ctx, *req.GroupId)
		if err != nil {
			return nil, err
		}

		group, ok := model.(*models.Group)
		if !ok {
			return nil, errors.New("group with id %s not found", *req.GroupId, errors.WithErrorCode(errors.ENotFound))
		}
		input.Group = group
	}

	result, err := s.serviceCatalog.TerraformModuleRegistryService.GetModules(ctx, input)
	if err != nil {
		return nil, err
	}

	modules := result.Modules

	pbModules := make([]*pb.TerraformModule, len(modules))
	for ix := range modules {
		pbModules[ix] = toPBTerraformModule(&modules[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(modules) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&modules[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&modules[len(modules)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTerraformModulesResponse{
		PageInfo: pageInfo,
		Modules:  pbModules,
	}, nil
}

// CreateTerraformModule creates a new TerraformModule.
func (s *TerraformModuleServer) CreateTerraformModule(ctx context.Context, req *pb.CreateTerraformModuleRequest) (*pb.TerraformModule, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	input := &moduleregistry.CreateModuleInput{
		Name:          req.Name,
		System:        req.System,
		GroupID:       groupID,
		RepositoryURL: req.RepositoryUrl,
		Private:       req.Private,
	}

	createdModule, err := s.serviceCatalog.TerraformModuleRegistryService.CreateModule(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTerraformModule(createdModule), nil
}

// UpdateTerraformModule returns the updated TerraformModule.
func (s *TerraformModuleServer) UpdateTerraformModule(ctx context.Context, req *pb.UpdateTerraformModuleRequest) (*pb.TerraformModule, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	module, ok := model.(*models.TerraformModule)
	if !ok {
		return nil, errors.New("terraform module with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		module.Metadata.Version = int(*req.Version)
	}

	if req.RepositoryUrl != nil {
		module.RepositoryURL = *req.RepositoryUrl
	}

	if req.Private != nil {
		module.Private = *req.Private
	}

	updatedModule, err := s.serviceCatalog.TerraformModuleRegistryService.UpdateModule(ctx, module)
	if err != nil {
		return nil, err
	}

	return toPBTerraformModule(updatedModule), nil
}

// DeleteTerraformModule deletes a TerraformModule.
func (s *TerraformModuleServer) DeleteTerraformModule(ctx context.Context, req *pb.DeleteTerraformModuleRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	module, ok := model.(*models.TerraformModule)
	if !ok {
		return nil, errors.New("terraform module with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformModuleRegistryService.DeleteModule(ctx, module); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetTerraformModuleVersionByID returns a TerraformModuleVersion by ID.
func (s *TerraformModuleServer) GetTerraformModuleVersionByID(ctx context.Context, req *pb.GetTerraformModuleVersionByIDRequest) (*pb.TerraformModuleVersion, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	version, ok := model.(*models.TerraformModuleVersion)
	if !ok {
		return nil, errors.New("terraform module version with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformModuleVersion(version), nil
}

// GetTerraformModuleVersions returns a paginated list of TerraformModuleVersions.
func (s *TerraformModuleServer) GetTerraformModuleVersions(ctx context.Context, req *pb.GetTerraformModuleVersionsRequest) (*pb.GetTerraformModuleVersionsResponse, error) {
	sort := db.TerraformModuleVersionSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	moduleID, err := s.serviceCatalog.FetchModelID(ctx, req.ModuleId)
	if err != nil {
		return nil, err
	}

	input := &moduleregistry.GetModuleVersionsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		ModuleID:          moduleID,
	}

	result, err := s.serviceCatalog.TerraformModuleRegistryService.GetModuleVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	versions := result.ModuleVersions

	pbVersions := make([]*pb.TerraformModuleVersion, len(versions))
	for ix := range versions {
		pbVersions[ix] = toPBTerraformModuleVersion(&versions[ix])
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

	return &pb.GetTerraformModuleVersionsResponse{
		PageInfo: pageInfo,
		Versions: pbVersions,
	}, nil
}

// CreateTerraformModuleVersion creates a new TerraformModuleVersion.
func (s *TerraformModuleServer) CreateTerraformModuleVersion(ctx context.Context, req *pb.CreateTerraformModuleVersionRequest) (*pb.TerraformModuleVersion, error) {
	moduleID, err := s.serviceCatalog.FetchModelID(ctx, req.ModuleId)
	if err != nil {
		return nil, err
	}

	shaSum, err := hex.DecodeString(req.ShaSum)
	if err != nil {
		return nil, errors.Wrap(err, "invalid sha_sum hex string", errors.WithErrorCode(errors.EInvalid))
	}

	input := &moduleregistry.CreateModuleVersionInput{
		ModuleID:        moduleID,
		SemanticVersion: req.Version,
		SHASum:          shaSum,
	}

	version, err := s.serviceCatalog.TerraformModuleRegistryService.CreateModuleVersion(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTerraformModuleVersion(version), nil
}

// DeleteTerraformModuleVersion deletes a TerraformModuleVersion.
func (s *TerraformModuleServer) DeleteTerraformModuleVersion(ctx context.Context, req *pb.DeleteTerraformModuleVersionRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	version, ok := model.(*models.TerraformModuleVersion)
	if !ok {
		return nil, errors.New("terraform module version with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformModuleRegistryService.DeleteModuleVersion(ctx, version); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetTerraformModuleAttestationByID returns a TerraformModuleAttestation by ID.
func (s *TerraformModuleServer) GetTerraformModuleAttestationByID(ctx context.Context, req *pb.GetTerraformModuleAttestationByIDRequest) (*pb.TerraformModuleAttestation, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	attestation, ok := model.(*models.TerraformModuleAttestation)
	if !ok {
		return nil, errors.New("terraform module attestation with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformModuleAttestation(attestation), nil
}

// GetTerraformModuleAttestations returns a paginated list of TerraformModuleAttestations.
func (s *TerraformModuleServer) GetTerraformModuleAttestations(ctx context.Context, req *pb.GetTerraformModuleAttestationsRequest) (*pb.GetTerraformModuleAttestationsResponse, error) {
	sort := db.TerraformModuleAttestationSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	moduleID, err := s.serviceCatalog.FetchModelID(ctx, req.ModuleId)
	if err != nil {
		return nil, err
	}

	input := &moduleregistry.GetModuleAttestationsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		ModuleID:          moduleID,
	}

	if req.Digest != nil {
		input.Digest = req.Digest
	}

	result, err := s.serviceCatalog.TerraformModuleRegistryService.GetModuleAttestations(ctx, input)
	if err != nil {
		return nil, err
	}

	attestations := result.ModuleAttestations

	pbAttestations := make([]*pb.TerraformModuleAttestation, len(attestations))
	for ix := range attestations {
		pbAttestations[ix] = toPBTerraformModuleAttestation(&attestations[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(attestations) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&attestations[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&attestations[len(attestations)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTerraformModuleAttestationsResponse{
		PageInfo:     pageInfo,
		Attestations: pbAttestations,
	}, nil
}

// CreateTerraformModuleAttestation creates a new TerraformModuleAttestation.
func (s *TerraformModuleServer) CreateTerraformModuleAttestation(ctx context.Context, req *pb.CreateTerraformModuleAttestationRequest) (*pb.TerraformModuleAttestation, error) {
	moduleID, err := s.serviceCatalog.FetchModelID(ctx, req.ModuleId)
	if err != nil {
		return nil, err
	}

	input := &moduleregistry.CreateModuleAttestationInput{
		ModuleID:        moduleID,
		Description:     req.Description,
		AttestationData: req.AttestationData,
	}

	attestation, err := s.serviceCatalog.TerraformModuleRegistryService.CreateModuleAttestation(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTerraformModuleAttestation(attestation), nil
}

// UpdateTerraformModuleAttestation updates a TerraformModuleAttestation.
func (s *TerraformModuleServer) UpdateTerraformModuleAttestation(ctx context.Context, req *pb.UpdateTerraformModuleAttestationRequest) (*pb.TerraformModuleAttestation, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	attestation, ok := model.(*models.TerraformModuleAttestation)
	if !ok {
		return nil, errors.New("terraform module attestation with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Description != nil {
		attestation.Description = *req.Description
	}

	updated, err := s.serviceCatalog.TerraformModuleRegistryService.UpdateModuleAttestation(ctx, attestation)
	if err != nil {
		return nil, err
	}

	return toPBTerraformModuleAttestation(updated), nil
}

// DeleteTerraformModuleAttestation deletes a TerraformModuleAttestation.
func (s *TerraformModuleServer) DeleteTerraformModuleAttestation(ctx context.Context, req *pb.DeleteTerraformModuleAttestationRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	attestation, ok := model.(*models.TerraformModuleAttestation)
	if !ok {
		return nil, errors.New("terraform module attestation with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformModuleRegistryService.DeleteModuleAttestation(ctx, attestation); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBTerraformModule converts from TerraformModule model to ProtoBuf model.
func toPBTerraformModule(m *models.TerraformModule) *pb.TerraformModule {
	return &pb.TerraformModule{
		Metadata:      toPBMetadata(&m.Metadata, types.TerraformModuleModelType),
		Name:          m.Name,
		System:        m.System,
		GroupId:       gid.ToGlobalID(types.GroupModelType, m.GroupID),
		RepositoryUrl: m.RepositoryURL,
		Private:       m.Private,
		CreatedBy:     m.CreatedBy,
	}
}

// toPBTerraformModuleVersion converts from TerraformModuleVersion model to ProtoBuf model.
func toPBTerraformModuleVersion(v *models.TerraformModuleVersion) *pb.TerraformModuleVersion {
	return &pb.TerraformModuleVersion{
		Metadata:        toPBMetadata(&v.Metadata, types.TerraformModuleVersionModelType),
		ModuleId:        gid.ToGlobalID(types.TerraformModuleModelType, v.ModuleID),
		SemanticVersion: v.SemanticVersion,
		Status:          string(v.Status),
		ShaSum:          v.GetSHASumHex(),
		Submodules:      v.Submodules,
		Latest:          v.Latest,
		CreatedBy:       v.CreatedBy,
	}
}

// toPBTerraformModuleAttestation converts from TerraformModuleAttestation model to ProtoBuf model.
func toPBTerraformModuleAttestation(a *models.TerraformModuleAttestation) *pb.TerraformModuleAttestation {
	return &pb.TerraformModuleAttestation{
		Metadata:      toPBMetadata(&a.Metadata, types.TerraformModuleAttestationModelType),
		ModuleId:      gid.ToGlobalID(types.TerraformModuleModelType, a.ModuleID),
		Description:   a.Description,
		Data:          a.Data,
		SchemaType:    a.SchemaType,
		PredicateType: a.PredicateType,
		Digests:       a.Digests,
		CreatedBy:     a.CreatedBy,
	}
}
