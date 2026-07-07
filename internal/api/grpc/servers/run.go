// Package servers implements the gRPC servers.
package servers

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/aws/smithy-go/ptr"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RunServer embeds the UnimplementedRunsServer.
type RunServer struct {
	pb.UnimplementedRunsServer
	serviceCatalog *services.Catalog
}

// NewRunServer returns an instance of RunServer.
func NewRunServer(serviceCatalog *services.Catalog) *RunServer {
	return &RunServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetRunByID returns a Run by an ID.
func (s *RunServer) GetRunByID(ctx context.Context, req *pb.GetRunByIDRequest) (*pb.Run, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	runModel, ok := model.(*models.Run)
	if !ok {
		return nil, errors.New("run with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBRun(runModel), nil
}

// GetRuns returns a paginated list of Runs.
func (s *RunServer) GetRuns(ctx context.Context, req *pb.GetRunsRequest) (*pb.GetRunsResponse, error) {
	sort := db.RunSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &run.GetRunsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		IncludeNestedRuns: req.IncludeNestedRuns,
	}

	if req.WorkspaceId != nil {
		model, wErr := s.serviceCatalog.FetchModel(ctx, *req.WorkspaceId)
		if wErr != nil {
			return nil, wErr
		}
		workspace, ok := model.(*models.Workspace)
		if !ok {
			return nil, errors.New("workspace with id %s not found", *req.WorkspaceId, errors.WithErrorCode(errors.ENotFound))
		}
		input.Workspace = workspace
	}

	if req.GroupId != nil {
		model, gErr := s.serviceCatalog.FetchModel(ctx, *req.GroupId)
		if gErr != nil {
			return nil, gErr
		}
		group, ok := model.(*models.Group)
		if !ok {
			return nil, errors.New("group with id %s not found", *req.GroupId, errors.WithErrorCode(errors.ENotFound))
		}
		input.Group = group
	}

	result, err := s.serviceCatalog.RunService.GetRuns(ctx, input)
	if err != nil {
		return nil, err
	}

	runs := result.Runs

	pbRuns := make([]*pb.Run, len(runs))
	for ix := range runs {
		pbRuns[ix] = toPBRun(runs[ix])
	}

	totalCount, err := result.PageInfo.TotalCount(ctx)
	if err != nil {
		return nil, err
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      totalCount,
	}

	if len(runs) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(runs[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(runs[len(runs)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetRunsResponse{
		PageInfo: pageInfo,
		Runs:     pbRuns,
	}, nil
}

// CreateRun creates a new Run.
func (s *RunServer) CreateRun(ctx context.Context, req *pb.CreateRunRequest) (*pb.Run, error) {
	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	var cvID *string
	if req.ConfigurationVersionId != nil {
		id, cvErr := s.serviceCatalog.FetchModelID(ctx, *req.ConfigurationVersionId)
		if cvErr != nil {
			return nil, cvErr
		}
		cvID = &id
	}

	variables := make([]runvariables.Variable, len(req.Variables))
	for i, v := range req.Variables {
		variables[i] = runvariables.Variable{
			Category: models.VariableCategory(v.Category),
			Key:      v.Key,
			Value:    v.Value,
		}
	}

	toCreate := &run.CreateRunInput{
		WorkspaceID:            workspaceID,
		ConfigurationVersionID: cvID,
		IsDestroy:              req.IsDestroy,
		TerraformVersion:       req.GetTerraformVersion(),
		Speculative:            req.Speculative,
		TargetAddresses:        req.TargetAddresses,
		// proto3 bool can't express "unset"; always pass an explicit value (gRPC
		// keeps its current behavior — omitted refresh stays false).
		Refresh:                  ptr.Bool(req.Refresh),
		RefreshOnly:              req.RefreshOnly,
		Variables:                variables,
		ModuleSource:             req.ModuleSource,
		ModuleVersion:            req.ModuleVersion,
		IncludeModulePrereleases: req.GetIncludeModulePrereleases(),
		AutoApply:                req.GetAutoApply(),
	}

	createdRun, err := s.serviceCatalog.RunService.CreateRun(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBRun(createdRun), nil
}

// ApplyRun applies a Run.
func (s *RunServer) ApplyRun(ctx context.Context, req *pb.ApplyRunRequest) (*pb.Run, error) {
	runID, err := s.serviceCatalog.FetchModelID(ctx, req.RunId)
	if err != nil {
		return nil, err
	}

	appliedRun, err := s.serviceCatalog.RunService.ApplyRun(ctx, runID, nil)
	if err != nil {
		return nil, err
	}

	return toPBRun(appliedRun), nil
}

// CancelRun cancels a Run.
func (s *RunServer) CancelRun(ctx context.Context, req *pb.CancelRunRequest) (*pb.Run, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	runModel, ok := model.(*models.Run)
	if !ok {
		return nil, errors.New("run with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	input := &run.CancelRunInput{
		RunID: runModel.Metadata.ID,
		Force: req.GetForce(),
	}

	canceledRun, err := s.serviceCatalog.RunService.CancelRun(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBRun(canceledRun), nil
}

// CreateDestroyRunForWorkspace creates a destroy run using the workspace's current state.
func (s *RunServer) CreateDestroyRunForWorkspace(ctx context.Context, req *pb.CreateDestroyRunForWorkspaceRequest) (*pb.Run, error) {
	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	createdRun, err := s.serviceCatalog.RunService.CreateDestroyRunForWorkspace(ctx, &run.CreateDestroyRunForWorkspaceInput{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	return toPBRun(createdRun), nil
}

// GetRunVariables returns variables for a Run.
func (s *RunServer) GetRunVariables(ctx context.Context, req *pb.GetRunVariablesRequest) (*pb.GetRunVariablesResponse, error) {
	runID, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	variables, err := s.serviceCatalog.RunService.GetRunVariables(ctx, runID, req.IncludeSensitiveValues)
	if err != nil {
		return nil, err
	}

	pbVariables := make([]*pb.RunVariable, len(variables))
	for i, v := range variables {
		pbVariables[i] = &pb.RunVariable{
			NamespacePath:      v.NamespacePath,
			Category:           string(v.Category),
			Key:                v.Key,
			Value:              v.Value,
			Sensitive:          v.Sensitive,
			VersionId:          v.VersionID,
			IncludedInTfConfig: v.IncludedInTFConfig != nil && *v.IncludedInTFConfig,
		}
	}

	return &pb.GetRunVariablesResponse{
		Variables: pbVariables,
	}, nil
}

// GetPlanByID returns a Plan by an ID.
func (s *RunServer) GetPlanByID(ctx context.Context, req *pb.GetPlanByIDRequest) (*pb.Plan, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	run, ok := model.(*models.Run)
	if !ok {
		return nil, errors.New("plan with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBPlan(run), nil
}

// GetApplyByID returns an Apply by an ID.
func (s *RunServer) GetApplyByID(ctx context.Context, req *pb.GetApplyByIDRequest) (*pb.Apply, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	run, ok := model.(*models.Run)
	if !ok {
		return nil, errors.New("apply with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if run.Apply == nil {
		return nil, errors.New("apply with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBApply(run), nil
}

// UpdatePlan updates a Plan.
func (s *RunServer) UpdatePlan(ctx context.Context, req *pb.UpdatePlanRequest) (*pb.Plan, error) {
	planID, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &run.UpdatePlanInput{
		PlanID:     planID,
		HasChanges: req.HasChanges,
	}

	if req.Version != nil {
		input.MetadataVersion = ptr.Int(int(*req.Version))
	}

	if req.ErrorMessage != nil {
		input.ErrorMessage = req.ErrorMessage
	}

	updatedPlan, err := s.serviceCatalog.RunService.UpdatePlan(ctx, input)
	if err != nil {
		return nil, err
	}

	run, err := s.serviceCatalog.RunService.GetRunByNodeID(ctx, updatedPlan.ID)
	if err != nil {
		return nil, err
	}

	return toPBPlan(run), nil
}

// UpdateApply updates an Apply.
func (s *RunServer) UpdateApply(ctx context.Context, req *pb.UpdateApplyRequest) (*pb.Apply, error) {
	applyID, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &run.UpdateApplyInput{
		ApplyID: applyID,
	}

	if req.Version != nil {
		input.MetadataVersion = ptr.Int(int(*req.Version))
	}

	if req.ErrorMessage != nil {
		input.ErrorMessage = req.ErrorMessage
	}

	updatedApply, err := s.serviceCatalog.RunService.UpdateApply(ctx, input)
	if err != nil {
		return nil, err
	}

	run, err := s.serviceCatalog.RunService.GetRunByNodeID(ctx, updatedApply.ID)
	if err != nil {
		return nil, err
	}

	if run.Apply == nil {
		return nil, errors.New("apply with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBApply(run), nil
}

// SetVariablesIncludedInTFConfig updates which variables are included in the Terraform config.
func (s *RunServer) SetVariablesIncludedInTFConfig(ctx context.Context, req *pb.SetVariablesIncludedInTFConfigRequest) (*emptypb.Empty, error) {
	runID, err := s.serviceCatalog.FetchModelID(ctx, req.RunId)
	if err != nil {
		return nil, err
	}

	if err = s.serviceCatalog.RunService.SetVariablesIncludedInTFConfig(ctx, &run.SetVariablesIncludedInTFConfigInput{
		RunID:        runID,
		VariableKeys: req.VariableKeys,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// SubscribeToRunEvents subscribes to run events.
func (s *RunServer) SubscribeToRunEvents(req *pb.SubscribeToRunEventsRequest, stream pb.Runs_SubscribeToRunEventsServer) error {
	var workspaceID, runID, ancestorGroupID *string

	if req.WorkspaceId != nil {
		id, err := s.serviceCatalog.FetchModelID(stream.Context(), *req.WorkspaceId)
		if err != nil {
			return err
		}
		workspaceID = &id
	}

	if req.RunId != nil {
		id, err := s.serviceCatalog.FetchModelID(stream.Context(), *req.RunId)
		if err != nil {
			return err
		}
		runID = &id
	}

	if req.AncestorGroupId != nil {
		id, err := s.serviceCatalog.FetchModelID(stream.Context(), *req.AncestorGroupId)
		if err != nil {
			return err
		}
		ancestorGroupID = &id
	}

	options := &run.EventSubscriptionOptions{
		WorkspaceID:     workspaceID,
		RunID:           runID,
		AncestorGroupID: ancestorGroupID,
	}

	eventChan, err := s.serviceCatalog.RunService.SubscribeToRunEvents(stream.Context(), options)
	if err != nil {
		return err
	}

	for event := range eventChan {
		pbEvent := &pb.RunEvent{
			Action: event.Action,
			Run:    toPBRun(event.Run),
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// toPBRunStatus maps a model RunStatus to its protobuf enum value. Model status
// strings are the lowercase form of the proto enum names, so an unknown value
// falls back to the zero value (UNSPECIFIED).
func toPBRunStatus(s models.RunStatus) pb.RunStatus {
	return pb.RunStatus(pb.RunStatus_value[strings.ToUpper(string(s))])
}

// toPBPlanStatus maps a model PlanStatus to its protobuf enum value.
func toPBPlanStatus(s models.PlanStatus) pb.PlanStatus {
	return pb.PlanStatus(pb.PlanStatus_value[strings.ToUpper(string(s))])
}

// toPBApplyStatus maps a model ApplyStatus to its protobuf enum value.
func toPBApplyStatus(s models.ApplyStatus) pb.ApplyStatus {
	return pb.ApplyStatus(pb.ApplyStatus_value[strings.ToUpper(string(s))])
}

// toPBRun converts from Run model to ProtoBuf model.
func toPBRun(r *models.Run) *pb.Run {
	pbRun := &pb.Run{
		Metadata:         toPBMetadata(&r.Metadata, types.RunModelType),
		CreatedBy:        r.CreatedBy,
		ForceCanceled:    r.ForceCanceled,
		ForceCanceledBy:  r.ForceCanceledBy,
		HasChanges:       r.HasChanges(),
		IsDestroy:        r.IsDestroy,
		Speculative:      r.Speculative(),
		AutoApply:        r.AutoApply,
		ModuleSource:     r.ModuleSource,
		ModuleVersion:    r.ModuleVersion,
		Refresh:          r.Refresh,
		RefreshOnly:      r.RefreshOnly,
		DeprecatedStatus: string(r.Status),
		Status:           toPBRunStatus(r.Status),
		TargetAddresses:  r.TargetAddresses,
		TerraformVersion: r.TerraformVersion,
		WorkspaceId:      gid.ToGlobalID(types.WorkspaceModelType, r.WorkspaceID),
		Plan:             toPBPlan(r),
		Apply:            toPBApply(r),
	}

	pbRun.PlanId = r.Plan.GetGlobalID()

	if applyNode := r.Apply; applyNode != nil {
		pbRun.ApplyId = applyNode.GetGlobalID()
	}

	if r.ModuleDigest != nil {
		pbRun.ModuleDigest = new(hex.EncodeToString(r.ModuleDigest))
	}

	if r.ConfigurationVersionID != nil {
		cvID := gid.ToGlobalID(types.ConfigurationVersionModelType, *r.ConfigurationVersionID)
		pbRun.ConfigurationVersionId = &cvID
	}

	if r.ForceCancelAvailableAt != nil {
		pbRun.ForceCancelAvailableAt = timestamppb.New(*r.ForceCancelAvailableAt)
	}

	return pbRun
}

// toPBPlan converts from Plan model to ProtoBuf model.
func toPBPlan(run *models.Run) *pb.Plan {
	p := run.Plan
	pbPlan := &pb.Plan{
		Metadata:         toPBMetadata(p.Metadata(run), types.PlanModelType),
		DeprecatedStatus: string(p.Status),
		Status:           toPBPlanStatus(p.Status),
		HasChanges:       p.HasChanges,
		ErrorMessage:     p.ErrorMessage,
		Summary: &pb.PlanSummary{
			ResourceAdditions:    p.Summary.ResourceAdditions,
			ResourceChanges:      p.Summary.ResourceChanges,
			ResourceDestructions: p.Summary.ResourceDestructions,
		},
	}
	if p.LatestJobID != nil {
		jobID := gid.ToGlobalID(types.JobModelType, *p.LatestJobID)
		pbPlan.LatestJobId = &jobID
	}
	return pbPlan
}

// toPBApply converts from Apply model to ProtoBuf model. It returns nil for a
// speculative run, which has no apply node.
func toPBApply(run *models.Run) *pb.Apply {
	a := run.Apply
	if a == nil {
		return nil
	}
	pbApply := &pb.Apply{
		Metadata:         toPBMetadata(a.Metadata(run), types.ApplyModelType),
		DeprecatedStatus: string(a.Status),
		Status:           toPBApplyStatus(a.Status),
		TriggeredBy:      a.TriggeredBy,
		ErrorMessage:     a.ErrorMessage,
	}
	if a.LatestJobID != nil {
		jobID := gid.ToGlobalID(types.JobModelType, *a.LatestJobID)
		pbApply.LatestJobId = &jobID
	}
	return pbApply
}
