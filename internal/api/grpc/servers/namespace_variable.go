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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// NamespaceVariableServer embeds the UnimplementedNamespaceVariablesServer.
type NamespaceVariableServer struct {
	pb.UnimplementedNamespaceVariablesServer
	serviceCatalog *services.Catalog
}

// NewNamespaceVariableServer returns an instance of NamespaceVariableServer.
func NewNamespaceVariableServer(serviceCatalog *services.Catalog) *NamespaceVariableServer {
	return &NamespaceVariableServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetNamespaceVariableByID returns a NamespaceVariable by an ID.
func (s *NamespaceVariableServer) GetNamespaceVariableByID(ctx context.Context, req *pb.GetNamespaceVariableByIDRequest) (*pb.NamespaceVariable, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	variable, ok := model.(*models.Variable)
	if !ok {
		return nil, errors.New("namespace variable with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBNamespaceVariable(variable), nil
}

// GetNamespaceVariables returns a list of NamespaceVariables for a namespace.
func (s *NamespaceVariableServer) GetNamespaceVariables(ctx context.Context, req *pb.GetNamespaceVariablesRequest) (*pb.GetNamespaceVariablesResponse, error) {
	variables, err := s.serviceCatalog.VariableService.GetVariables(ctx, req.NamespacePath)
	if err != nil {
		return nil, err
	}

	pbVariables := make([]*pb.NamespaceVariable, len(variables))
	for ix := range variables {
		pbVariables[ix] = toPBNamespaceVariable(&variables[ix])
	}

	return &pb.GetNamespaceVariablesResponse{
		Variables: pbVariables,
	}, nil
}

// CreateNamespaceVariable creates a new NamespaceVariable.
func (s *NamespaceVariableServer) CreateNamespaceVariable(ctx context.Context, req *pb.CreateNamespaceVariableRequest) (*pb.NamespaceVariable, error) {
	input := &variable.CreateVariableInput{
		Category:      models.VariableCategory(strings.ToLower(req.Category.String())),
		Key:           req.Key,
		NamespacePath: req.NamespacePath,
		Sensitive:     req.Sensitive,
		Value:         req.Value,
	}

	createdVariable, err := s.serviceCatalog.VariableService.CreateVariable(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBNamespaceVariable(createdVariable), nil
}

// UpdateNamespaceVariable returns the updated NamespaceVariable.
func (s *NamespaceVariableServer) UpdateNamespaceVariable(ctx context.Context, req *pb.UpdateNamespaceVariableRequest) (*pb.NamespaceVariable, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &variable.UpdateVariableInput{
		ID:    id,
		Key:   req.Key,
		Value: req.Value,
	}

	updatedVariable, err := s.serviceCatalog.VariableService.UpdateVariable(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBNamespaceVariable(updatedVariable), nil
}

// DeleteNamespaceVariable deletes a NamespaceVariable.
func (s *NamespaceVariableServer) DeleteNamespaceVariable(ctx context.Context, req *pb.DeleteNamespaceVariableRequest) (*emptypb.Empty, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &variable.DeleteVariableInput{
		ID: id,
	}

	if req.Version != nil {
		version := int(*req.Version)
		input.MetadataVersion = &version
	}

	if err := s.serviceCatalog.VariableService.DeleteVariable(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// SetNamespaceVariables sets multiple variables for a namespace.
func (s *NamespaceVariableServer) SetNamespaceVariables(ctx context.Context, req *pb.SetNamespaceVariablesRequest) (*emptypb.Empty, error) {
	variables := make([]*variable.SetVariablesInputVariable, len(req.Variables))
	for i, v := range req.Variables {
		variables[i] = &variable.SetVariablesInputVariable{
			Key:       v.Key,
			Sensitive: v.Sensitive,
			Value:     v.Value,
		}
	}

	input := &variable.SetVariablesInput{
		Category:      models.VariableCategory(strings.ToLower(req.Category.String())),
		NamespacePath: req.NamespacePath,
		Variables:     variables,
	}

	if err := s.serviceCatalog.VariableService.SetVariables(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetNamespaceVariableVersionByID returns a NamespaceVariableVersion by an ID.
func (s *NamespaceVariableServer) GetNamespaceVariableVersionByID(ctx context.Context, req *pb.GetNamespaceVariableVersionByIDRequest) (*pb.NamespaceVariableVersion, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	variableVersion, ok := model.(*models.VariableVersion)
	if !ok {
		return nil, errors.New("namespace variable version with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBNamespaceVariableVersion(variableVersion), nil
}

// GetNamespaceVariableVersions returns a list of NamespaceVariableVersions.
func (s *NamespaceVariableServer) GetNamespaceVariableVersions(ctx context.Context, req *pb.GetNamespaceVariableVersionsRequest) (*pb.GetNamespaceVariableVersionsResponse, error) {
	variableID, err := s.serviceCatalog.FetchModelID(ctx, req.VariableId)
	if err != nil {
		return nil, err
	}

	sort := db.VariableVersionSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &variable.GetVariableVersionsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		VariableID:        variableID,
	}

	result, err := s.serviceCatalog.VariableService.GetVariableVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	versions := result.VariableVersions

	pbVersions := make([]*pb.NamespaceVariableVersion, len(versions))
	for ix := range versions {
		pbVersions[ix] = toPBNamespaceVariableVersion(&versions[ix])
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

	return &pb.GetNamespaceVariableVersionsResponse{
		PageInfo:                  pageInfo,
		NamespaceVariableVersions: pbVersions,
	}, nil
}

// toPBNamespaceVariable converts from Variable model to ProtoBuf model.
func toPBNamespaceVariable(v *models.Variable) *pb.NamespaceVariable {
	return &pb.NamespaceVariable{
		Metadata:        toPBMetadata(&v.Metadata, types.VariableModelType),
		Category:        string(v.Category),
		Key:             v.Key,
		NamespacePath:   v.NamespacePath,
		Sensitive:       v.Sensitive,
		Value:           v.Value,
		LatestVersionId: gid.ToGlobalID(types.VariableVersionModelType, v.LatestVersionID),
	}
}

// toPBNamespaceVariableVersion converts from VariableVersion model to ProtoBuf model.
func toPBNamespaceVariableVersion(vv *models.VariableVersion) *pb.NamespaceVariableVersion {
	return &pb.NamespaceVariableVersion{
		Metadata:            toPBMetadata(&vv.Metadata, types.VariableVersionModelType),
		Key:                 vv.Key,
		Value:               vv.Value,
		NamespaceVariableId: gid.ToGlobalID(types.VariableModelType, vv.VariableID),
	}
}
