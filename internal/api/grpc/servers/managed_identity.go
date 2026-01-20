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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ManagedIdentityServer embeds the UnimplementedManagedIdentitiesServer.
type ManagedIdentityServer struct {
	pb.UnimplementedManagedIdentitiesServer
	serviceCatalog *services.Catalog
}

// NewManagedIdentityServer returns an instance of ManagedIdentityServer.
func NewManagedIdentityServer(serviceCatalog *services.Catalog) *ManagedIdentityServer {
	return &ManagedIdentityServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetManagedIdentityByID returns a ManagedIdentity by an ID.
func (s *ManagedIdentityServer) GetManagedIdentityByID(ctx context.Context, req *pb.GetManagedIdentityByIDRequest) (*pb.ManagedIdentity, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	managedIdentity, ok := model.(*models.ManagedIdentity)
	if !ok {
		return nil, errors.New("managed identity with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBManagedIdentity(managedIdentity), nil
}

// GetManagedIdentities returns a paginated list of ManagedIdentities.
func (s *ManagedIdentityServer) GetManagedIdentities(ctx context.Context, req *pb.GetManagedIdentitiesRequest) (*pb.GetManagedIdentitiesResponse, error) {
	sort := db.ManagedIdentitySortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &managedidentity.GetManagedIdentitiesInput{
		Search:            req.Search,
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		NamespacePath:     req.NamespacePath,
		IncludeInherited:  req.IncludeInherited,
	}

	if req.AliasSourceId != nil {
		aliasSourceID, aErr := s.serviceCatalog.FetchModelID(ctx, *req.AliasSourceId)
		if aErr != nil {
			return nil, aErr
		}
		input.AliasSourceID = &aliasSourceID
	}

	result, err := s.serviceCatalog.ManagedIdentityService.GetManagedIdentities(ctx, input)
	if err != nil {
		return nil, err
	}

	managedIdentities := result.ManagedIdentities

	pbManagedIdentities := make([]*pb.ManagedIdentity, len(managedIdentities))
	for ix := range managedIdentities {
		pbManagedIdentities[ix] = toPBManagedIdentity(&managedIdentities[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(managedIdentities) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&managedIdentities[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&managedIdentities[len(managedIdentities)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetManagedIdentitiesResponse{
		PageInfo:          pageInfo,
		ManagedIdentities: pbManagedIdentities,
	}, nil
}

// CreateManagedIdentity creates a new ManagedIdentity.
func (s *ManagedIdentityServer) CreateManagedIdentity(ctx context.Context, req *pb.CreateManagedIdentityRequest) (*pb.ManagedIdentity, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	accessRules := make([]struct {
		Type                      models.ManagedIdentityAccessRuleType
		RunStage                  models.JobType
		ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
		AllowedUserIDs            []string
		AllowedServiceAccountIDs  []string
		AllowedTeamIDs            []string
		VerifyStateLineage        bool
	}, len(req.AccessRules))

	for i, rule := range req.AccessRules {
		allowedUserIDs, err := s.resolvePrincipalIDs(ctx, rule.AllowedUsers)
		if err != nil {
			return nil, err
		}

		allowedServiceAccountIDs, err := s.resolvePrincipalIDs(ctx, rule.AllowedServiceAccounts)
		if err != nil {
			return nil, err
		}

		allowedTeamIDs, err := s.resolvePrincipalIDs(ctx, rule.AllowedTeams)
		if err != nil {
			return nil, err
		}

		accessRules[i] = struct {
			Type                      models.ManagedIdentityAccessRuleType
			RunStage                  models.JobType
			ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
			AllowedUserIDs            []string
			AllowedServiceAccountIDs  []string
			AllowedTeamIDs            []string
			VerifyStateLineage        bool
		}{
			Type:                     models.ManagedIdentityAccessRuleType(strings.ToLower(rule.Type.String())),
			RunStage:                 models.JobType(strings.ToLower(rule.RunStage.String())),
			AllowedUserIDs:           allowedUserIDs,
			AllowedServiceAccountIDs: allowedServiceAccountIDs,
			AllowedTeamIDs:           allowedTeamIDs,
			VerifyStateLineage:       rule.VerifyStateLineage,
		}

		for _, policy := range rule.ModuleAttestationPolicies {
			accessRules[i].ModuleAttestationPolicies = append(accessRules[i].ModuleAttestationPolicies,
				models.ManagedIdentityAccessRuleModuleAttestationPolicy{
					PredicateType: policy.PredicateType,
					PublicKey:     policy.PublicKey,
				})
		}
	}

	toCreate := &managedidentity.CreateManagedIdentityInput{
		Type:        models.ManagedIdentityType(strings.ToLower(req.Type.String())),
		Name:        req.Name,
		Description: req.Description,
		GroupID:     groupID,
		Data:        []byte(req.Data),
		AccessRules: accessRules,
	}

	createdManagedIdentity, err := s.serviceCatalog.ManagedIdentityService.CreateManagedIdentity(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBManagedIdentity(createdManagedIdentity), nil
}

// UpdateManagedIdentity returns the updated ManagedIdentity.
func (s *ManagedIdentityServer) UpdateManagedIdentity(ctx context.Context, req *pb.UpdateManagedIdentityRequest) (*pb.ManagedIdentity, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	toUpdate := &managedidentity.UpdateManagedIdentityInput{
		ID: id,
	}

	if req.Description != nil {
		toUpdate.Description = req.Description
	}

	if req.Data != nil {
		toUpdate.Data = []byte(*req.Data)
	}

	updatedManagedIdentity, err := s.serviceCatalog.ManagedIdentityService.UpdateManagedIdentity(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	return toPBManagedIdentity(updatedManagedIdentity), nil
}

// DeleteManagedIdentity deletes a ManagedIdentity.
func (s *ManagedIdentityServer) DeleteManagedIdentity(ctx context.Context, req *pb.DeleteManagedIdentityRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotManagedIdentity, ok := model.(*models.ManagedIdentity)
	if !ok {
		return nil, errors.New("managed identity with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	toDelete := &managedidentity.DeleteManagedIdentityInput{
		ManagedIdentity: gotManagedIdentity,
		Force:           req.GetForce(),
	}

	if err := s.serviceCatalog.ManagedIdentityService.DeleteManagedIdentity(ctx, toDelete); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateManagedIdentityCredentials creates credentials for a ManagedIdentity.
func (s *ManagedIdentityServer) CreateManagedIdentityCredentials(ctx context.Context, req *pb.CreateManagedIdentityCredentialsRequest) (*pb.CreateManagedIdentityCredentialsResponse, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.ManagedIdentityId)
	if err != nil {
		return nil, err
	}

	managedIdentity, ok := model.(*models.ManagedIdentity)
	if !ok {
		return nil, errors.New("managed identity with id %s not found", req.ManagedIdentityId, errors.WithErrorCode(errors.ENotFound))
	}

	data, err := s.serviceCatalog.ManagedIdentityService.CreateCredentials(ctx, managedIdentity)
	if err != nil {
		return nil, err
	}

	return &pb.CreateManagedIdentityCredentialsResponse{
		Data: string(data),
	}, nil
}

// AssignManagedIdentityToWorkspace assigns a ManagedIdentity to a Workspace.
func (s *ManagedIdentityServer) AssignManagedIdentityToWorkspace(ctx context.Context, req *pb.AssignManagedIdentityToWorkspaceRequest) (*emptypb.Empty, error) {
	managedIdentityID, err := s.serviceCatalog.FetchModelID(ctx, req.ManagedIdentityId)
	if err != nil {
		return nil, err
	}

	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.ManagedIdentityService.AddManagedIdentityToWorkspace(ctx, managedIdentityID, workspaceID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetManagedIdentitiesForWorkspace returns ManagedIdentities for a Workspace.
func (s *ManagedIdentityServer) GetManagedIdentitiesForWorkspace(ctx context.Context, req *pb.GetManagedIdentitiesForWorkspaceRequest) (*pb.GetManagedIdentitiesResponse, error) {
	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	managedIdentities, err := s.serviceCatalog.ManagedIdentityService.GetManagedIdentitiesForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	pbManagedIdentities := make([]*pb.ManagedIdentity, len(managedIdentities))
	for ix := range managedIdentities {
		pbManagedIdentities[ix] = toPBManagedIdentity(&managedIdentities[ix])
	}

	return &pb.GetManagedIdentitiesResponse{
		ManagedIdentities: pbManagedIdentities,
	}, nil
}

// RemoveManagedIdentityFromWorkspace removes a ManagedIdentity from a Workspace.
func (s *ManagedIdentityServer) RemoveManagedIdentityFromWorkspace(ctx context.Context, req *pb.RemoveManagedIdentityFromWorkspaceRequest) (*emptypb.Empty, error) {
	managedIdentityID, err := s.serviceCatalog.FetchModelID(ctx, req.ManagedIdentityId)
	if err != nil {
		return nil, err
	}

	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.ManagedIdentityService.RemoveManagedIdentityFromWorkspace(ctx, managedIdentityID, workspaceID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetManagedIdentityAccessRules returns access rules for a ManagedIdentity.
func (s *ManagedIdentityServer) GetManagedIdentityAccessRules(ctx context.Context, req *pb.GetManagedIdentityAccessRulesRequest) (*pb.GetManagedIdentityAccessRulesResponse, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.ManagedIdentityId)
	if err != nil {
		return nil, err
	}

	managedIdentity, ok := model.(*models.ManagedIdentity)
	if !ok {
		return nil, errors.New("managed identity with id %s not found", req.ManagedIdentityId, errors.WithErrorCode(errors.ENotFound))
	}

	accessRules, err := s.serviceCatalog.ManagedIdentityService.GetManagedIdentityAccessRules(ctx, managedIdentity)
	if err != nil {
		return nil, err
	}

	pbAccessRules := make([]*pb.ManagedIdentityAccessRule, len(accessRules))
	for ix := range accessRules {
		pbAccessRules[ix] = toPBManagedIdentityAccessRule(&accessRules[ix])
	}

	return &pb.GetManagedIdentityAccessRulesResponse{
		AccessRules: pbAccessRules,
	}, nil
}

// GetManagedIdentityAccessRuleByID returns an access rule by ID.
func (s *ManagedIdentityServer) GetManagedIdentityAccessRuleByID(ctx context.Context, req *pb.GetManagedIdentityAccessRuleByIDRequest) (*pb.ManagedIdentityAccessRule, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	accessRule, err := s.serviceCatalog.ManagedIdentityService.GetManagedIdentityAccessRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toPBManagedIdentityAccessRule(accessRule), nil
}

// CreateManagedIdentityAccessRule creates a new access rule.
func (s *ManagedIdentityServer) CreateManagedIdentityAccessRule(ctx context.Context, req *pb.CreateManagedIdentityAccessRuleRequest) (*pb.ManagedIdentityAccessRule, error) {
	managedIdentityID, err := s.serviceCatalog.FetchModelID(ctx, req.ManagedIdentityId)
	if err != nil {
		return nil, err
	}

	allowedUserIDs, err := s.resolvePrincipalIDs(ctx, req.AllowedUsers)
	if err != nil {
		return nil, err
	}

	allowedServiceAccountIDs, err := s.resolvePrincipalIDs(ctx, req.AllowedServiceAccounts)
	if err != nil {
		return nil, err
	}

	allowedTeamIDs, err := s.resolvePrincipalIDs(ctx, req.AllowedTeams)
	if err != nil {
		return nil, err
	}

	moduleAttestationPolicies := make([]models.ManagedIdentityAccessRuleModuleAttestationPolicy, len(req.ModuleAttestationPolicies))
	for i, policy := range req.ModuleAttestationPolicies {
		moduleAttestationPolicies[i] = models.ManagedIdentityAccessRuleModuleAttestationPolicy{
			PredicateType: policy.PredicateType,
			PublicKey:     policy.PublicKey,
		}
	}

	toCreate := &models.ManagedIdentityAccessRule{
		Type:                      models.ManagedIdentityAccessRuleType(strings.ToLower(req.Type.String())),
		RunStage:                  models.JobType(strings.ToLower(req.RunStage.String())),
		ManagedIdentityID:         managedIdentityID,
		AllowedUserIDs:            allowedUserIDs,
		AllowedServiceAccountIDs:  allowedServiceAccountIDs,
		AllowedTeamIDs:            allowedTeamIDs,
		VerifyStateLineage:        req.VerifyStateLineage,
		ModuleAttestationPolicies: moduleAttestationPolicies,
	}

	createdAccessRule, err := s.serviceCatalog.ManagedIdentityService.CreateManagedIdentityAccessRule(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBManagedIdentityAccessRule(createdAccessRule), nil
}

// UpdateManagedIdentityAccessRule updates an access rule.
func (s *ManagedIdentityServer) UpdateManagedIdentityAccessRule(ctx context.Context, req *pb.UpdateManagedIdentityAccessRuleRequest) (*pb.ManagedIdentityAccessRule, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	rule, ok := model.(*models.ManagedIdentityAccessRule)
	if !ok {
		return nil, errors.New("managed identity access rule with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if len(req.AllowedUsers) > 0 {
		rule.AllowedUserIDs, err = s.resolvePrincipalIDs(ctx, req.AllowedUsers)
		if err != nil {
			return nil, err
		}
	}

	if len(req.AllowedServiceAccounts) > 0 {
		rule.AllowedServiceAccountIDs, err = s.resolvePrincipalIDs(ctx, req.AllowedServiceAccounts)
		if err != nil {
			return nil, err
		}
	}

	if len(req.AllowedTeams) > 0 {
		rule.AllowedTeamIDs, err = s.resolvePrincipalIDs(ctx, req.AllowedTeams)
		if err != nil {
			return nil, err
		}
	}

	if len(req.ModuleAttestationPolicies) > 0 {
		rule.ModuleAttestationPolicies = make([]models.ManagedIdentityAccessRuleModuleAttestationPolicy, len(req.ModuleAttestationPolicies))
		for i, policy := range req.ModuleAttestationPolicies {
			rule.ModuleAttestationPolicies[i] = models.ManagedIdentityAccessRuleModuleAttestationPolicy{
				PredicateType: policy.PredicateType,
				PublicKey:     policy.PublicKey,
			}
		}
	}

	rule.RunStage = models.JobType(strings.ToLower(req.RunStage.String()))

	if req.VerifyStateLineage != nil {
		rule.VerifyStateLineage = *req.VerifyStateLineage
	}

	updatedAccessRule, err := s.serviceCatalog.ManagedIdentityService.UpdateManagedIdentityAccessRule(ctx, rule)
	if err != nil {
		return nil, err
	}

	return toPBManagedIdentityAccessRule(updatedAccessRule), nil
}

// DeleteManagedIdentityAccessRule deletes an access rule.
func (s *ManagedIdentityServer) DeleteManagedIdentityAccessRule(ctx context.Context, req *pb.DeleteManagedIdentityAccessRuleRequest) (*emptypb.Empty, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	accessRule, err := s.serviceCatalog.ManagedIdentityService.GetManagedIdentityAccessRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.ManagedIdentityService.DeleteManagedIdentityAccessRule(ctx, accessRule); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateManagedIdentityAlias creates an alias for a ManagedIdentity.
func (s *ManagedIdentityServer) CreateManagedIdentityAlias(ctx context.Context, req *pb.CreateManagedIdentityAliasRequest) (*pb.ManagedIdentity, error) {
	aliasSourceID, err := s.serviceCatalog.FetchModelID(ctx, req.AliasSourceId)
	if err != nil {
		return nil, err
	}

	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	toCreate := &managedidentity.CreateManagedIdentityAliasInput{
		Name:          req.Name,
		AliasSourceID: aliasSourceID,
		GroupID:       groupID,
	}

	createdAlias, err := s.serviceCatalog.ManagedIdentityService.CreateManagedIdentityAlias(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBManagedIdentity(createdAlias), nil
}

// DeleteManagedIdentityAlias deletes a ManagedIdentity alias.
func (s *ManagedIdentityServer) DeleteManagedIdentityAlias(ctx context.Context, req *pb.DeleteManagedIdentityAliasRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotManagedIdentity, ok := model.(*models.ManagedIdentity)
	if !ok {
		return nil, errors.New("managed identity with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	toDelete := &managedidentity.DeleteManagedIdentityInput{
		ManagedIdentity: gotManagedIdentity,
		Force:           req.GetForce(),
	}

	if err := s.serviceCatalog.ManagedIdentityService.DeleteManagedIdentityAlias(ctx, toDelete); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// MoveManagedIdentity moves a ManagedIdentity to a new group.
func (s *ManagedIdentityServer) MoveManagedIdentity(ctx context.Context, req *pb.MoveManagedIdentityRequest) (*pb.ManagedIdentity, error) {
	managedIdentityID, err := s.serviceCatalog.FetchModelID(ctx, req.ManagedIdentityId)
	if err != nil {
		return nil, err
	}

	newGroupID, err := s.serviceCatalog.FetchModelID(ctx, req.NewGroupId)
	if err != nil {
		return nil, err
	}

	toMove := &managedidentity.MoveManagedIdentityInput{
		ManagedIdentityID: managedIdentityID,
		NewGroupID:        newGroupID,
	}

	movedManagedIdentity, err := s.serviceCatalog.ManagedIdentityService.MoveManagedIdentity(ctx, toMove)
	if err != nil {
		return nil, err
	}

	return toPBManagedIdentity(movedManagedIdentity), nil
}

// toPBManagedIdentity converts from ManagedIdentity model to ProtoBuf model.
func toPBManagedIdentity(m *models.ManagedIdentity) *pb.ManagedIdentity {
	resp := &pb.ManagedIdentity{
		Metadata:    toPBMetadata(&m.Metadata, types.ManagedIdentityModelType),
		Name:        m.Name,
		Description: m.Description,
		GroupId:     gid.ToGlobalID(types.GroupModelType, m.GroupID),
		Type:        string(m.Type),
		Data:        string(m.Data),
		CreatedBy:   m.CreatedBy,
	}

	if m.AliasSourceID != nil {
		aliasSourceID := gid.ToGlobalID(types.ManagedIdentityModelType, *m.AliasSourceID)
		resp.AliasSourceId = &aliasSourceID
	}

	return resp
}

// toPBManagedIdentityAccessRule converts from ManagedIdentityAccessRule model to ProtoBuf model.
func toPBManagedIdentityAccessRule(r *models.ManagedIdentityAccessRule) *pb.ManagedIdentityAccessRule {
	moduleAttestationPolicies := make([]*pb.ManagedIdentityAccessRuleModuleAttestationPolicy, len(r.ModuleAttestationPolicies))
	for i, policy := range r.ModuleAttestationPolicies {
		moduleAttestationPolicies[i] = &pb.ManagedIdentityAccessRuleModuleAttestationPolicy{
			PredicateType: policy.PredicateType,
			PublicKey:     policy.PublicKey,
		}
	}

	return &pb.ManagedIdentityAccessRule{
		Metadata:                  toPBMetadata(&r.Metadata, types.ManagedIdentityAccessRuleModelType),
		Type:                      string(r.Type),
		RunStage:                  string(r.RunStage),
		ManagedIdentityId:         gid.ToGlobalID(types.ManagedIdentityModelType, r.ManagedIdentityID),
		AllowedUsers:              r.AllowedUserIDs,
		AllowedServiceAccounts:    r.AllowedServiceAccountIDs,
		AllowedTeams:              r.AllowedTeamIDs,
		VerifyStateLineage:        r.VerifyStateLineage,
		ModuleAttestationPolicies: moduleAttestationPolicies,
	}
}

func (s *ManagedIdentityServer) resolvePrincipalIDs(ctx context.Context, ids []string) ([]string, error) {
	result := make([]string, len(ids))
	for i, id := range ids {
		resolved, err := s.serviceCatalog.FetchModelID(ctx, id)
		if err != nil {
			return nil, err
		}
		result[i] = resolved
	}
	return result, nil
}
