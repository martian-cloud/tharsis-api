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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TerraformProviderMirrorServer embeds the UnimplementedTerraformProviderMirrorsServer.
type TerraformProviderMirrorServer struct {
	pb.UnimplementedTerraformProviderMirrorsServer
	serviceCatalog *services.Catalog
}

// NewTerraformProviderMirrorServer returns an instance of TerraformProviderMirrorServer.
func NewTerraformProviderMirrorServer(serviceCatalog *services.Catalog) *TerraformProviderMirrorServer {
	return &TerraformProviderMirrorServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetTerraformProviderVersionMirrorByID returns a version mirror by ID.
func (s *TerraformProviderMirrorServer) GetTerraformProviderVersionMirrorByID(ctx context.Context, req *pb.GetTerraformProviderVersionMirrorByIDRequest) (*pb.TerraformProviderVersionMirror, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	versionMirror, ok := model.(*models.TerraformProviderVersionMirror)
	if !ok {
		return nil, errors.New("terraform provider version mirror with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformProviderVersionMirror(versionMirror), nil
}

// GetTerraformProviderVersionMirrors returns a paginated list of version mirrors.
func (s *TerraformProviderMirrorServer) GetTerraformProviderVersionMirrors(ctx context.Context, req *pb.GetTerraformProviderVersionMirrorsRequest) (*pb.GetTerraformProviderVersionMirrorsResponse, error) {
	sort := db.TerraformProviderVersionMirrorSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &providermirror.GetProviderVersionMirrorsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		NamespacePath:     req.NamespacePath,
	}

	result, err := s.serviceCatalog.TerraformProviderMirrorService.GetProviderVersionMirrors(ctx, input)
	if err != nil {
		return nil, err
	}

	mirrors := result.VersionMirrors

	pbMirrors := make([]*pb.TerraformProviderVersionMirror, len(mirrors))
	for ix := range mirrors {
		pbMirrors[ix] = toPBTerraformProviderVersionMirror(&mirrors[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(mirrors) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&mirrors[0])
		if err != nil {
			return nil, err
		}
		pageInfo.EndCursor, err = result.PageInfo.Cursor(&mirrors[len(mirrors)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTerraformProviderVersionMirrorsResponse{
		PageInfo:       pageInfo,
		VersionMirrors: pbMirrors,
	}, nil
}

// CreateTerraformProviderVersionMirror creates a new version mirror.
func (s *TerraformProviderMirrorServer) CreateTerraformProviderVersionMirror(ctx context.Context, req *pb.CreateTerraformProviderVersionMirrorRequest) (*pb.TerraformProviderVersionMirror, error) {
	input := &providermirror.CreateProviderVersionMirrorInput{
		GroupPath:         req.GroupPath,
		Type:              req.Type,
		RegistryNamespace: req.RegistryNamespace,
		RegistryHostname:  req.RegistryHostname,
		SemanticVersion:   req.SemanticVersion,
		RegistryToken:     req.RegistryToken,
	}

	created, err := s.serviceCatalog.TerraformProviderMirrorService.CreateProviderVersionMirror(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTerraformProviderVersionMirror(created), nil
}

// DeleteTerraformProviderVersionMirror deletes a version mirror.
func (s *TerraformProviderMirrorServer) DeleteTerraformProviderVersionMirror(ctx context.Context, req *pb.DeleteTerraformProviderVersionMirrorRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	versionMirror, ok := model.(*models.TerraformProviderVersionMirror)
	if !ok {
		return nil, errors.New("terraform provider version mirror with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformProviderMirrorService.DeleteProviderVersionMirror(ctx, &providermirror.DeleteProviderVersionMirrorInput{
		VersionMirror: versionMirror,
		Force:         req.Force,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetTerraformProviderPlatformMirrorByID returns a platform mirror by ID.
func (s *TerraformProviderMirrorServer) GetTerraformProviderPlatformMirrorByID(ctx context.Context, req *pb.GetTerraformProviderPlatformMirrorByIDRequest) (*pb.TerraformProviderPlatformMirror, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	platformMirror, ok := model.(*models.TerraformProviderPlatformMirror)
	if !ok {
		return nil, errors.New("terraform provider platform mirror with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTerraformProviderPlatformMirror(platformMirror), nil
}

// GetTerraformProviderPlatformMirrors returns a paginated list of platform mirrors.
func (s *TerraformProviderMirrorServer) GetTerraformProviderPlatformMirrors(ctx context.Context, req *pb.GetTerraformProviderPlatformMirrorsRequest) (*pb.GetTerraformProviderPlatformMirrorsResponse, error) {
	sort := db.TerraformProviderPlatformMirrorSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	versionMirrorID, err := s.serviceCatalog.FetchModelID(ctx, req.VersionMirrorId)
	if err != nil {
		return nil, err
	}

	input := &providermirror.GetProviderPlatformMirrorsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		VersionMirrorID:   versionMirrorID,
		OS:                req.Os,
		Architecture:      req.Architecture,
	}

	result, err := s.serviceCatalog.TerraformProviderMirrorService.GetProviderPlatformMirrors(ctx, input)
	if err != nil {
		return nil, err
	}

	mirrors := result.PlatformMirrors

	pbMirrors := make([]*pb.TerraformProviderPlatformMirror, len(mirrors))
	for ix := range mirrors {
		pbMirrors[ix] = toPBTerraformProviderPlatformMirror(&mirrors[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(mirrors) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&mirrors[0])
		if err != nil {
			return nil, err
		}
		pageInfo.EndCursor, err = result.PageInfo.Cursor(&mirrors[len(mirrors)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTerraformProviderPlatformMirrorsResponse{
		PageInfo:        pageInfo,
		PlatformMirrors: pbMirrors,
	}, nil
}

// DeleteTerraformProviderPlatformMirror deletes a platform mirror.
func (s *TerraformProviderMirrorServer) DeleteTerraformProviderPlatformMirror(ctx context.Context, req *pb.DeleteTerraformProviderPlatformMirrorRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	platformMirror, ok := model.(*models.TerraformProviderPlatformMirror)
	if !ok {
		return nil, errors.New("terraform provider platform mirror with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.TerraformProviderMirrorService.DeleteProviderPlatformMirror(ctx, &providermirror.DeleteProviderPlatformMirrorInput{
		PlatformMirror: platformMirror,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func toPBTerraformProviderVersionMirror(m *models.TerraformProviderVersionMirror) *pb.TerraformProviderVersionMirror {
	digests := make(map[string]string, len(m.Digests))
	for k, v := range m.Digests {
		digests[k] = hex.EncodeToString(v)
	}

	return &pb.TerraformProviderVersionMirror{
		Metadata:          toPBMetadata(&m.Metadata, types.TerraformProviderVersionMirrorModelType),
		CreatedBy:         m.CreatedBy,
		Type:              m.Type,
		SemanticVersion:   m.SemanticVersion,
		RegistryNamespace: m.RegistryNamespace,
		RegistryHostname:  m.RegistryHostname,
		Digests:           digests,
		GroupId:           gid.ToGlobalID(types.GroupModelType, m.GroupID),
	}
}

func toPBTerraformProviderPlatformMirror(m *models.TerraformProviderPlatformMirror) *pb.TerraformProviderPlatformMirror {
	return &pb.TerraformProviderPlatformMirror{
		Metadata:        toPBMetadata(&m.Metadata, types.TerraformProviderPlatformMirrorModelType),
		Os:              m.OS,
		Architecture:    m.Architecture,
		VersionMirrorId: gid.ToGlobalID(types.TerraformProviderVersionMirrorModelType, m.VersionMirrorID),
	}
}
