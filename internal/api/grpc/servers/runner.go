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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RunnerServer embeds the UnimplementedRunnersServer.
type RunnerServer struct {
	pb.UnimplementedRunnersServer
	serviceCatalog *services.Catalog
}

// NewRunnerServer returns an instance of RunnerServer.
func NewRunnerServer(serviceCatalog *services.Catalog) *RunnerServer {
	return &RunnerServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetRunnerByID returns a Runner by an ID.
func (s *RunnerServer) GetRunnerByID(ctx context.Context, req *pb.GetRunnerByIDRequest) (*pb.Runner, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	runner, ok := model.(*models.Runner)
	if !ok {
		return nil, errors.New("runner with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBRunner(runner), nil
}

// GetRunners returns a paginated list of Runners.
func (s *RunnerServer) GetRunners(ctx context.Context, req *pb.GetRunnersRequest) (*pb.GetRunnersResponse, error) {
	sort := db.RunnerSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	var runnerType *models.RunnerType
	if req.RunnerType != nil {
		rt := models.RunnerType(strings.ToLower(req.RunnerType.String()))
		runnerType = &rt
	}

	input := &runner.GetRunnersInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		NamespacePath:     req.NamespacePath,
		RunnerType:        runnerType,
		IncludeInherited:  req.IncludeInherited,
	}

	result, err := s.serviceCatalog.RunnerService.GetRunners(ctx, input)
	if err != nil {
		return nil, err
	}

	runners := result.Runners

	pbRunners := make([]*pb.Runner, len(runners))
	for ix := range runners {
		pbRunners[ix] = toPBRunner(&runners[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(runners) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&runners[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&runners[len(runners)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetRunnersResponse{
		PageInfo: pageInfo,
		Runners:  pbRunners,
	}, nil
}

// CreateRunner creates a new Runner.
func (s *RunnerServer) CreateRunner(ctx context.Context, req *pb.CreateRunnerRequest) (*pb.Runner, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	input := &runner.CreateRunnerInput{
		Name:            req.Name,
		Description:     req.Description,
		GroupID:         groupID,
		Disabled:        req.Disabled,
		RunUntaggedJobs: req.RunUntaggedJobs,
		Tags:            req.Tags,
	}

	createdRunner, err := s.serviceCatalog.RunnerService.CreateRunner(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBRunner(createdRunner), nil
}

// UpdateRunner returns the updated Runner.
func (s *RunnerServer) UpdateRunner(ctx context.Context, req *pb.UpdateRunnerRequest) (*pb.Runner, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotRunner, ok := model.(*models.Runner)
	if !ok {
		return nil, errors.New("runner with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotRunner.Metadata.Version = int(*req.Version)
	}

	if req.Description != nil {
		gotRunner.Description = *req.Description
	}

	if req.Disabled != nil {
		gotRunner.Disabled = *req.Disabled
	}

	if req.RunUntaggedJobs != nil {
		gotRunner.RunUntaggedJobs = *req.RunUntaggedJobs
	}

	if len(req.Tags) > 0 {
		gotRunner.Tags = req.Tags
	}

	updatedRunner, err := s.serviceCatalog.RunnerService.UpdateRunner(ctx, gotRunner)
	if err != nil {
		return nil, err
	}

	return toPBRunner(updatedRunner), nil
}

// DeleteRunner deletes a Runner.
func (s *RunnerServer) DeleteRunner(ctx context.Context, req *pb.DeleteRunnerRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotRunner, ok := model.(*models.Runner)
	if !ok {
		return nil, errors.New("runner with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotRunner.Metadata.Version = int(*req.Version)
	}

	if err := s.serviceCatalog.RunnerService.DeleteRunner(ctx, gotRunner); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// AssignServiceAccountToRunner assigns a service account to a runner.
func (s *RunnerServer) AssignServiceAccountToRunner(ctx context.Context, req *pb.AssignServiceAccountToRunnerRequest) (*emptypb.Empty, error) {
	runnerID, err := s.serviceCatalog.FetchModelID(ctx, req.RunnerId)
	if err != nil {
		return nil, err
	}

	serviceAccountID, err := s.serviceCatalog.FetchModelID(ctx, req.ServiceAccountId)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.RunnerService.AssignServiceAccountToRunner(ctx, serviceAccountID, runnerID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// UnassignServiceAccountFromRunner unassigns a service account from a runner.
func (s *RunnerServer) UnassignServiceAccountFromRunner(ctx context.Context, req *pb.UnassignServiceAccountFromRunnerRequest) (*emptypb.Empty, error) {
	runnerID, err := s.serviceCatalog.FetchModelID(ctx, req.RunnerId)
	if err != nil {
		return nil, err
	}

	serviceAccountID, err := s.serviceCatalog.FetchModelID(ctx, req.ServiceAccountId)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.RunnerService.UnassignServiceAccountFromRunner(ctx, serviceAccountID, runnerID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateRunnerSession creates a new runner session.
func (s *RunnerServer) CreateRunnerSession(ctx context.Context, req *pb.CreateRunnerSessionRequest) (*pb.RunnerSession, error) {
	input := &runner.CreateRunnerSessionInput{
		RunnerPath: req.RunnerPath,
	}

	session, err := s.serviceCatalog.RunnerService.CreateRunnerSession(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBRunnerSession(session), nil
}

// GetRunnerSessions returns a paginated list of runner sessions.
func (s *RunnerServer) GetRunnerSessions(ctx context.Context, req *pb.GetRunnerSessionsRequest) (*pb.GetRunnerSessionsResponse, error) {
	sort := db.RunnerSessionSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	runnerID, err := s.serviceCatalog.FetchModelID(ctx, req.RunnerId)
	if err != nil {
		return nil, err
	}

	input := &runner.GetRunnerSessionsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		RunnerID:          runnerID,
	}

	result, err := s.serviceCatalog.RunnerService.GetRunnerSessions(ctx, input)
	if err != nil {
		return nil, err
	}

	sessions := result.RunnerSessions

	pbSessions := make([]*pb.RunnerSession, len(sessions))
	for ix := range sessions {
		pbSessions[ix] = toPBRunnerSession(&sessions[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(sessions) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&sessions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&sessions[len(sessions)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetRunnerSessionsResponse{
		PageInfo:       pageInfo,
		RunnerSessions: pbSessions,
	}, nil
}

// GetRunnerSessionByID returns a runner session by ID.
func (s *RunnerServer) GetRunnerSessionByID(ctx context.Context, req *pb.GetRunnerSessionByIDRequest) (*pb.RunnerSession, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	session, ok := model.(*models.RunnerSession)
	if !ok {
		return nil, errors.New("runner session with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBRunnerSession(session), nil
}

// SendRunnerSessionHeartbeat accepts a heartbeat from a runner session.
func (s *RunnerServer) SendRunnerSessionHeartbeat(ctx context.Context, req *pb.SendRunnerSessionHeartbeatRequest) (*emptypb.Empty, error) {
	sessionID, err := s.serviceCatalog.FetchModelID(ctx, req.SessionId)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.RunnerService.AcceptRunnerSessionHeartbeat(ctx, sessionID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateRunnerSessionError creates an error for a runner session.
func (s *RunnerServer) CreateRunnerSessionError(ctx context.Context, req *pb.CreateRunnerSessionErrorRequest) (*emptypb.Empty, error) {
	sessionID, err := s.serviceCatalog.FetchModelID(ctx, req.RunnerSessionId)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.RunnerService.CreateRunnerSessionError(ctx, sessionID, req.Message); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// SubscribeToRunnerSessions subscribes to runner session events.
func (s *RunnerServer) SubscribeToRunnerSessions(req *pb.SubscribeToRunnerSessionsRequest, stream pb.Runners_SubscribeToRunnerSessionsServer) error {
	var groupID, runnerID *string
	var runnerType *models.RunnerType

	if req.GroupId != nil {
		id, err := s.serviceCatalog.FetchModelID(stream.Context(), *req.GroupId)
		if err != nil {
			return err
		}
		groupID = &id
	}

	if req.RunnerId != nil {
		id, err := s.serviceCatalog.FetchModelID(stream.Context(), *req.RunnerId)
		if err != nil {
			return err
		}
		runnerID = &id
	}

	if req.RunnerType != nil {
		rt := models.RunnerType(strings.ToLower(*req.RunnerType))
		runnerType = &rt
	}

	options := &runner.SubscribeToRunnerSessionsInput{
		GroupID:    groupID,
		RunnerID:   runnerID,
		RunnerType: runnerType,
	}

	eventChan, err := s.serviceCatalog.RunnerService.SubscribeToRunnerSessions(stream.Context(), options)
	if err != nil {
		return err
	}

	for event := range eventChan {
		pbEvent := &pb.RunnerSessionEvent{
			Action:        event.Action,
			RunnerSession: toPBRunnerSession(event.RunnerSession),
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// SubscribeToRunnerSessionErrorLog subscribes to runner session error log events.
func (s *RunnerServer) SubscribeToRunnerSessionErrorLog(req *pb.SubscribeToRunnerSessionErrorLogRequest, stream pb.Runners_SubscribeToRunnerSessionErrorLogServer) error {
	sessionID, err := s.serviceCatalog.FetchModelID(stream.Context(), req.RunnerSessionId)
	if err != nil {
		return err
	}

	options := &runner.SubscribeToRunnerSessionErrorLogInput{
		RunnerSessionID: sessionID,
	}

	if req.LastSeenLogSize != nil {
		lastSeenLogSize := int(*req.LastSeenLogSize)
		options.LastSeenLogSize = &lastSeenLogSize
	}

	eventChan, err := s.serviceCatalog.RunnerService.SubscribeToRunnerSessionErrorLog(stream.Context(), options)
	if err != nil {
		return err
	}

	for event := range eventChan {
		pbEvent := &pb.RunnerSessionErrorLogEvent{
			Completed: event.Completed,
			Size:      int32(event.Size),
		}

		if event.Data != nil {
			pbEvent.Data = &pb.RunnerSessionErrorLogEventData{
				Offset: int32(event.Data.Offset),
				Logs:   event.Data.Logs,
			}
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// toPBRunner converts from Runner model to ProtoBuf model.
func toPBRunner(r *models.Runner) *pb.Runner {
	resp := &pb.Runner{
		Metadata:        toPBMetadata(&r.Metadata, types.RunnerModelType),
		Name:            r.Name,
		Description:     r.Description,
		Type:            string(r.Type),
		CreatedBy:       r.CreatedBy,
		Disabled:        r.Disabled,
		RunUntaggedJobs: r.RunUntaggedJobs,
		Tags:            r.Tags,
	}

	if r.GroupID != nil {
		groupID := gid.ToGlobalID(types.GroupModelType, *r.GroupID)
		resp.GroupId = &groupID
	}

	return resp
}

// toPBRunnerSession converts from RunnerSession model to ProtoBuf model.
func toPBRunnerSession(s *models.RunnerSession) *pb.RunnerSession {
	return &pb.RunnerSession{
		Metadata:        toPBMetadata(&s.Metadata, types.RunnerSessionModelType),
		RunnerId:        gid.ToGlobalID(types.RunnerModelType, s.RunnerID),
		LastContactedAt: timestamppb.New(s.LastContactTimestamp),
		ErrorCount:      int32(s.ErrorCount),
		Internal:        s.Internal,
	}
}
