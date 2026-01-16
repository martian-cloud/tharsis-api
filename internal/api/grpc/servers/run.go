// Package servers implements the gRPC servers.
package servers

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
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
		pbRuns[ix] = toPBRun(&runs[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(runs) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&runs[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&runs[len(runs)-1])
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

	variables := make([]run.Variable, len(req.Variables))
	for i, v := range req.Variables {
		variables[i] = run.Variable{
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
		Refresh:                req.Refresh,
		RefreshOnly:            req.RefreshOnly,
		Variables:              variables,
		ModuleSource:           req.ModuleSource,
		ModuleVersion:          req.ModuleVersion,
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

	plan, ok := model.(*models.Plan)
	if !ok {
		return nil, errors.New("plan with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBPlan(plan), nil
}

// GetApplyByID returns an Apply by an ID.
func (s *RunServer) GetApplyByID(ctx context.Context, req *pb.GetApplyByIDRequest) (*pb.Apply, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	apply, ok := model.(*models.Apply)
	if !ok {
		return nil, errors.New("apply with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBApply(apply), nil
}

// UpdatePlan updates a Plan.
func (s *RunServer) UpdatePlan(ctx context.Context, req *pb.UpdatePlanRequest) (*pb.Plan, error) {
	planID, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &run.UpdatePlanInput{
		PlanID:     planID,
		Status:     models.PlanStatus(strings.ToLower(req.Status.String())),
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

	return toPBPlan(updatedPlan), nil
}

// UpdateApply updates an Apply.
func (s *RunServer) UpdateApply(ctx context.Context, req *pb.UpdateApplyRequest) (*pb.Apply, error) {
	applyID, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &run.UpdateApplyInput{
		ApplyID: applyID,
		Status:  models.ApplyStatus(strings.ToLower(req.Status.String())),
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

	return toPBApply(updatedApply), nil
}

// GetLatestJobForPlan returns the latest job for a Plan.
func (s *RunServer) GetLatestJobForPlan(ctx context.Context, req *pb.GetLatestJobForPlanRequest) (*pb.Job, error) {
	planID, err := s.serviceCatalog.FetchModelID(ctx, req.PlanId)
	if err != nil {
		return nil, err
	}

	job, err := s.serviceCatalog.RunService.GetLatestJobForPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	return toPBJob(job), nil
}

// GetLatestJobForApply returns the latest job for an Apply.
func (s *RunServer) GetLatestJobForApply(ctx context.Context, req *pb.GetLatestJobForApplyRequest) (*pb.Job, error) {
	applyID, err := s.serviceCatalog.FetchModelID(ctx, req.ApplyId)
	if err != nil {
		return nil, err
	}

	job, err := s.serviceCatalog.RunService.GetLatestJobForApply(ctx, applyID)
	if err != nil {
		return nil, err
	}

	return toPBJob(job), nil
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
			Run:    toPBRun(&event.Run),
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// toPBRun converts from Run model to ProtoBuf model.
func toPBRun(r *models.Run) *pb.Run {
	pbRun := &pb.Run{
		Metadata:         toPBMetadata(&r.Metadata, types.RunModelType),
		ApplyId:          gid.ToGlobalID(types.ApplyModelType, r.ApplyID),
		CreatedBy:        r.CreatedBy,
		ForceCanceled:    r.ForceCanceled,
		ForceCanceledBy:  r.ForceCanceledBy,
		HasChanges:       r.HasChanges,
		IsDestroy:        r.IsDestroy,
		ModuleDigest:     ptr.String(hex.EncodeToString(r.ModuleDigest)),
		ModuleSource:     r.ModuleSource,
		ModuleVersion:    r.ModuleVersion,
		PlanId:           gid.ToGlobalID(types.PlanModelType, r.PlanID),
		Refresh:          r.Refresh,
		RefreshOnly:      r.RefreshOnly,
		Status:           string(r.Status),
		TargetAddresses:  r.TargetAddresses,
		TerraformVersion: r.TerraformVersion,
		WorkspaceId:      gid.ToGlobalID(types.WorkspaceModelType, r.WorkspaceID),
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
func toPBPlan(p *models.Plan) *pb.Plan {
	return &pb.Plan{
		Metadata:     toPBMetadata(&p.Metadata, types.PlanModelType),
		Status:       string(p.Status),
		HasChanges:   p.HasChanges,
		ErrorMessage: p.ErrorMessage,
	}
}

// toPBApply converts from Apply model to ProtoBuf model.
func toPBApply(a *models.Apply) *pb.Apply {
	return &pb.Apply{
		Metadata:     toPBMetadata(&a.Metadata, types.ApplyModelType),
		Status:       string(a.Status),
		TriggeredBy:  a.TriggeredBy,
		ErrorMessage: a.ErrorMessage,
	}
}
