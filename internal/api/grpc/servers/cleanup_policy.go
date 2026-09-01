package servers

import (
	"context"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cleanuppolicy"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	pbCleanupRuleKindPrefix = "CLEANUP_RULE_KIND_"
	pbCleanupStrategyPrefix = "CLEANUP_STRATEGY_"
)

// CleanupPolicyServer embeds the UnimplementedCleanupPoliciesServer.
type CleanupPolicyServer struct {
	pb.UnimplementedCleanupPoliciesServer
	serviceCatalog *services.Catalog
}

// NewCleanupPolicyServer returns an instance of CleanupPolicyServer.
func NewCleanupPolicyServer(serviceCatalog *services.Catalog) *CleanupPolicyServer {
	return &CleanupPolicyServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetCleanupPolicyByID returns a cleanup policy by an ID.
func (s *CleanupPolicyServer) GetCleanupPolicyByID(ctx context.Context, req *pb.GetCleanupPolicyByIDRequest) (*pb.CleanupPolicy, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	policy, ok := model.(*models.CleanupPolicy)
	if !ok {
		return nil, errors.New("cleanup policy with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBCleanupPolicy(policy), nil
}

// GetEffectiveCleanupPolicies returns the effective cleanup policies for a namespace.
func (s *CleanupPolicyServer) GetEffectiveCleanupPolicies(ctx context.Context, req *pb.GetEffectiveCleanupPoliciesRequest) (*pb.GetEffectiveCleanupPoliciesResponse, error) {
	policies, err := s.serviceCatalog.CleanupPolicyService.GetEffectiveCleanupPolicies(ctx, req.NamespacePath)
	if err != nil {
		return nil, err
	}

	pbPolicies := make([]*pb.CleanupPolicy, len(policies))
	for ix := range policies {
		pbPolicies[ix] = toPBCleanupPolicy(policies[ix])
	}

	return &pb.GetEffectiveCleanupPoliciesResponse{
		Policies: pbPolicies,
	}, nil
}

// CreateCleanupPolicy creates a new cleanup policy.
func (s *CleanupPolicyServer) CreateCleanupPolicy(ctx context.Context, req *pb.CreateCleanupPolicyRequest) (*pb.CleanupPolicy, error) {
	input := &cleanuppolicy.CreateCleanupPolicyInput{
		NamespacePath:               req.NamespacePath,
		Kind:                        enumFromPB[models.CleanupRuleKind](req.Kind, pbCleanupRuleKindPrefix),
		TerraformModulePolicyData:   fromPBTerraformModuleCleanupPolicyData(req.TerraformModulePolicyData),
		TerraformProviderPolicyData: fromPBTerraformProviderCleanupPolicyData(req.TerraformProviderPolicyData),
		RunPolicyData:               fromPBRunCleanupPolicyData(req.RunPolicyData),
	}

	if req.Disabled != nil {
		input.Disabled = *req.Disabled
	}

	createdPolicy, err := s.serviceCatalog.CleanupPolicyService.CreateCleanupPolicy(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBCleanupPolicy(createdPolicy), nil
}

// UpdateCleanupPolicy returns the updated cleanup policy.
func (s *CleanupPolicyServer) UpdateCleanupPolicy(ctx context.Context, req *pb.UpdateCleanupPolicyRequest) (*pb.CleanupPolicy, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &cleanuppolicy.UpdateCleanupPolicyInput{
		ID:                          id,
		Disabled:                    req.Disabled,
		TerraformModulePolicyData:   fromPBTerraformModuleCleanupPolicyData(req.TerraformModulePolicyData),
		TerraformProviderPolicyData: fromPBTerraformProviderCleanupPolicyData(req.TerraformProviderPolicyData),
		RunPolicyData:               fromPBRunCleanupPolicyData(req.RunPolicyData),
	}

	if req.Version != nil {
		version := int(*req.Version)
		input.Version = &version
	}

	updatedPolicy, err := s.serviceCatalog.CleanupPolicyService.UpdateCleanupPolicy(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBCleanupPolicy(updatedPolicy), nil
}

// DeleteCleanupPolicy deletes a cleanup policy.
func (s *CleanupPolicyServer) DeleteCleanupPolicy(ctx context.Context, req *pb.DeleteCleanupPolicyRequest) (*emptypb.Empty, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &cleanuppolicy.DeleteCleanupPolicyInput{
		ID: id,
	}

	if req.Version != nil {
		version := int(*req.Version)
		input.Version = &version
	}

	if err := s.serviceCatalog.CleanupPolicyService.DeleteCleanupPolicy(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBCleanupPolicy converts from the CleanupPolicy model to its ProtoBuf equivalent.
func toPBCleanupPolicy(policy *models.CleanupPolicy) *pb.CleanupPolicy {
	return &pb.CleanupPolicy{
		Metadata:                    toPBMetadata(&policy.Metadata, types.CleanupPolicyModelType),
		Kind:                        enumToPB[pb.CleanupRuleKind](policy.Kind, pb.CleanupRuleKind_value, pbCleanupRuleKindPrefix),
		NamespacePath:               policy.NamespacePath(),
		Disabled:                    policy.Disabled,
		TerraformModulePolicyData:   toPBTerraformModuleCleanupPolicyData(policy.TerraformModulePolicyData),
		TerraformProviderPolicyData: toPBTerraformProviderCleanupPolicyData(policy.TerraformProviderPolicyData),
		RunPolicyData:               toPBRunCleanupPolicyData(policy.RunPolicyData),
	}
}

// toPBTerraformModuleCleanupPolicyData converts model module policy data to its ProtoBuf equivalent.
func toPBTerraformModuleCleanupPolicyData(data *models.TerraformModuleCleanupPolicyData) *pb.TerraformModuleCleanupPolicyData {
	if data == nil {
		return nil
	}

	pbRules := make([]*pb.TerraformModuleCleanupRule, len(data.Rules))
	for ix, rule := range data.Rules {
		pbRules[ix] = &pb.TerraformModuleCleanupRule{
			Strategy:        enumToPB[pb.CleanupStrategy](rule.Strategy, pb.CleanupStrategy_value, pbCleanupStrategyPrefix),
			Description:     rule.Description,
			NameGlob:        string(rule.NameGlob),
			SystemGlob:      string(rule.SystemGlob),
			VersionGlob:     string(rule.VersionGlob),
			DeleteAfterDays: rule.DeleteAfterDays,
		}
	}

	return &pb.TerraformModuleCleanupPolicyData{Rules: pbRules}
}

// toPBTerraformProviderCleanupPolicyData converts model provider policy data to its ProtoBuf equivalent.
func toPBTerraformProviderCleanupPolicyData(data *models.TerraformProviderCleanupPolicyData) *pb.TerraformProviderCleanupPolicyData {
	if data == nil {
		return nil
	}

	pbRules := make([]*pb.TerraformProviderCleanupRule, len(data.Rules))
	for ix, rule := range data.Rules {
		pbRules[ix] = &pb.TerraformProviderCleanupRule{
			Strategy:        enumToPB[pb.CleanupStrategy](rule.Strategy, pb.CleanupStrategy_value, pbCleanupStrategyPrefix),
			Description:     rule.Description,
			NameGlob:        string(rule.NameGlob),
			VersionGlob:     string(rule.VersionGlob),
			DeleteAfterDays: rule.DeleteAfterDays,
		}
	}

	return &pb.TerraformProviderCleanupPolicyData{Rules: pbRules}
}

// toPBRunCleanupPolicyData converts model run policy data to its ProtoBuf equivalent.
func toPBRunCleanupPolicyData(data *models.RunCleanupPolicyData) *pb.RunCleanupPolicyData {
	if data == nil {
		return nil
	}

	pbRules := make([]*pb.RunCleanupRule, len(data.Rules))
	for ix, rule := range data.Rules {
		statuses := make([]pb.RunStatus, len(rule.Status))
		for statusIx, status := range rule.Status {
			statuses[statusIx] = toPBRunStatus(status)
		}
		pbRules[ix] = &pb.RunCleanupRule{
			Strategy:        enumToPB[pb.CleanupStrategy](rule.Strategy, pb.CleanupStrategy_value, pbCleanupStrategyPrefix),
			Description:     rule.Description,
			Speculative:     rule.Speculative,
			Assessment:      rule.Assessment,
			Status:          statuses,
			KeepMin:         rule.KeepMin,
			DeleteAfterDays: rule.DeleteAfterDays,
		}
	}

	return &pb.RunCleanupPolicyData{Rules: pbRules}
}

// fromPBTerraformModuleCleanupPolicyData converts ProtoBuf module policy data to its model equivalent.
func fromPBTerraformModuleCleanupPolicyData(pbData *pb.TerraformModuleCleanupPolicyData) *models.TerraformModuleCleanupPolicyData {
	if pbData == nil {
		return nil
	}

	rules := make([]*models.TerraformModuleCleanupRule, len(pbData.Rules))
	for ix, pbRule := range pbData.Rules {
		rules[ix] = &models.TerraformModuleCleanupRule{
			Strategy:        enumFromPB[models.CleanupStrategy](pbRule.Strategy, pbCleanupStrategyPrefix),
			Description:     pbRule.Description,
			NameGlob:        models.CleanupGlob(pbRule.NameGlob),
			SystemGlob:      models.CleanupGlob(pbRule.SystemGlob),
			VersionGlob:     models.CleanupGlob(pbRule.VersionGlob),
			DeleteAfterDays: pbRule.DeleteAfterDays,
		}
	}

	return &models.TerraformModuleCleanupPolicyData{Rules: rules}
}

// fromPBTerraformProviderCleanupPolicyData converts ProtoBuf provider policy data to its model equivalent.
func fromPBTerraformProviderCleanupPolicyData(pbData *pb.TerraformProviderCleanupPolicyData) *models.TerraformProviderCleanupPolicyData {
	if pbData == nil {
		return nil
	}

	rules := make([]*models.TerraformProviderCleanupRule, len(pbData.Rules))
	for ix, pbRule := range pbData.Rules {
		rules[ix] = &models.TerraformProviderCleanupRule{
			Strategy:        enumFromPB[models.CleanupStrategy](pbRule.Strategy, pbCleanupStrategyPrefix),
			Description:     pbRule.Description,
			NameGlob:        models.CleanupGlob(pbRule.NameGlob),
			VersionGlob:     models.CleanupGlob(pbRule.VersionGlob),
			DeleteAfterDays: pbRule.DeleteAfterDays,
		}
	}

	return &models.TerraformProviderCleanupPolicyData{Rules: rules}
}

// fromPBRunCleanupPolicyData converts ProtoBuf run policy data to its model equivalent.
func fromPBRunCleanupPolicyData(pbData *pb.RunCleanupPolicyData) *models.RunCleanupPolicyData {
	if pbData == nil {
		return nil
	}

	rules := make([]*models.RunCleanupRule, len(pbData.Rules))
	for ix, pbRule := range pbData.Rules {
		statuses := make([]models.RunStatus, len(pbRule.Status))
		for statusIx, status := range pbRule.Status {
			statuses[statusIx] = models.RunStatus(strings.ToLower(status.String()))
		}

		rules[ix] = &models.RunCleanupRule{
			Strategy:        enumFromPB[models.CleanupStrategy](pbRule.Strategy, pbCleanupStrategyPrefix),
			Description:     pbRule.Description,
			Speculative:     pbRule.Speculative,
			Assessment:      pbRule.Assessment,
			Status:          statuses,
			KeepMin:         pbRule.KeepMin,
			DeleteAfterDays: pbRule.DeleteAfterDays,
		}
	}

	return &models.RunCleanupPolicyData{Rules: rules}
}
