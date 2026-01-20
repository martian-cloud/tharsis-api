// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GPGKeyServer embeds the UnimplementedGPGKeysServer.
type GPGKeyServer struct {
	pb.UnimplementedGPGKeysServer
	serviceCatalog *services.Catalog
}

// NewGPGKeyServer returns an instance of GPGKeyServer.
func NewGPGKeyServer(serviceCatalog *services.Catalog) *GPGKeyServer {
	return &GPGKeyServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetGPGKeyByID returns a GPG Key by an ID.
func (s *GPGKeyServer) GetGPGKeyByID(ctx context.Context, req *pb.GetGPGKeyByIDRequest) (*pb.GPGKey, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gpgKey, ok := model.(*models.GPGKey)
	if !ok {
		return nil, errors.New("gpg key with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBGPGKey(gpgKey), nil
}

// GetGPGKeys returns a paginated list of GPG Keys.
func (s *GPGKeyServer) GetGPGKeys(ctx context.Context, req *pb.GetGPGKeysRequest) (*pb.GetGPGKeysResponse, error) {
	sort := db.GPGKeySortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &gpgkey.GetGPGKeysInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		NamespacePath:     req.NamespacePath,
		IncludeInherited:  req.IncludeInherited,
	}

	result, err := s.serviceCatalog.GPGKeyService.GetGPGKeys(ctx, input)
	if err != nil {
		return nil, err
	}

	gpgKeys := result.GPGKeys

	pbGPGKeys := make([]*pb.GPGKey, len(gpgKeys))
	for ix := range gpgKeys {
		pbGPGKeys[ix] = toPBGPGKey(&gpgKeys[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(gpgKeys) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&gpgKeys[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&gpgKeys[len(gpgKeys)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetGPGKeysResponse{
		PageInfo: pageInfo,
		GpgKeys:  pbGPGKeys,
	}, nil
}

// CreateGPGKey creates a new GPG Key.
func (s *GPGKeyServer) CreateGPGKey(ctx context.Context, req *pb.CreateGPGKeyRequest) (*pb.GPGKey, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	input := &gpgkey.CreateGPGKeyInput{
		GroupID:    groupID,
		ASCIIArmor: req.AsciiArmor,
	}

	createdGPGKey, err := s.serviceCatalog.GPGKeyService.CreateGPGKey(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBGPGKey(createdGPGKey), nil
}

// DeleteGPGKey deletes a GPG Key.
func (s *GPGKeyServer) DeleteGPGKey(ctx context.Context, req *pb.DeleteGPGKeyRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gpgKey, ok := model.(*models.GPGKey)
	if !ok {
		return nil, errors.New("gpg key with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.serviceCatalog.GPGKeyService.DeleteGPGKey(ctx, gpgKey); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBGPGKey converts from GPGKey model to ProtoBuf model.
func toPBGPGKey(g *models.GPGKey) *pb.GPGKey {
	return &pb.GPGKey{
		Metadata:    toPBMetadata(&g.Metadata, types.GPGKeyModelType),
		GroupId:     gid.ToGlobalID(types.GroupModelType, g.GroupID),
		AsciiArmor:  g.ASCIIArmor,
		GpgKeyId:    g.GetHexGPGKeyID(),
		Fingerprint: g.Fingerprint,
		CreatedBy:   g.CreatedBy,
	}
}
