// Package servers implements the gRPC servers.
package servers

import (
	"context"
	"io"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// JobServer embeds the UnimplementedJobsServer.
type JobServer struct {
	pb.UnimplementedJobsServer
	serviceCatalog *services.Catalog
}

// NewJobServer returns an instance of JobServer.
func NewJobServer(serviceCatalog *services.Catalog) *JobServer {
	return &JobServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetJobByID retrieves a job by its ID.
func (s *JobServer) GetJobByID(ctx context.Context, req *pb.GetJobByIDRequest) (*pb.Job, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	job, ok := model.(*models.Job)
	if !ok {
		return nil, errors.New("job with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBJob(job), nil
}

// GetJobLogs retrieves job logs.
func (s *JobServer) GetJobLogs(ctx context.Context, req *pb.GetJobLogsRequest) (*pb.GetJobLogsResponse, error) {
	jobID, err := s.serviceCatalog.FetchModelID(ctx, req.JobId)
	if err != nil {
		return nil, err
	}

	reader, err := s.serviceCatalog.JobService.ReadLogs(ctx, jobID, int(req.StartOffset), int(req.Limit))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	logs, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return &pb.GetJobLogsResponse{
		Logs: string(logs),
	}, nil
}

// GetLatestJobForPlan retrieves the latest job for a plan ID.
func (s *JobServer) GetLatestJobForPlan(ctx context.Context, req *pb.GetLatestJobForPlanRequest) (*pb.Job, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.PlanId)
	if err != nil {
		return nil, err
	}

	run, ok := model.(*models.Run)
	if !ok {
		return nil, errors.New("expected run model, got %T", model)
	}

	if run.Plan.LatestJobID == nil {
		return nil, errors.New("plan with id %s does not have a latest job", req.PlanId, errors.WithErrorCode(errors.ENotFound))
	}

	job, err := s.serviceCatalog.JobService.GetJobByID(ctx, *run.Plan.LatestJobID)
	if err != nil {
		return nil, err
	}

	return toPBJob(job), nil
}

// GetLatestJobForApply retrieves the latest job for an apply ID.
func (s *JobServer) GetLatestJobForApply(ctx context.Context, req *pb.GetLatestJobForApplyRequest) (*pb.Job, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.ApplyId)
	if err != nil {
		return nil, err
	}

	run, ok := model.(*models.Run)
	if !ok {
		return nil, errors.New("expected run model, got %T", model)
	}

	apply := run.Apply
	if apply == nil {
		return nil, errors.New("apply with id %s not found", req.ApplyId, errors.WithErrorCode(errors.ENotFound))
	}

	if apply.LatestJobID == nil {
		return nil, errors.New("apply with id %s does not have a latest job", req.ApplyId, errors.WithErrorCode(errors.ENotFound))
	}

	job, err := s.serviceCatalog.JobService.GetJobByID(ctx, *apply.LatestJobID)
	if err != nil {
		return nil, err
	}

	return toPBJob(job), nil
}

// SetJobStatus sets the status of a job.
func (s *JobServer) SetJobStatus(ctx context.Context, req *pb.SetJobStatusInput) (*pb.Job, error) {
	jobID, err := s.serviceCatalog.FetchModelID(ctx, req.JobId)
	if err != nil {
		return nil, err
	}

	job, err := s.serviceCatalog.JobService.SetJobStatus(ctx, jobID, models.JobStatus(req.GetStatus().String()))
	if err != nil {
		return nil, err
	}

	return toPBJob(job), nil
}

// SaveJobLogs saves job logs.
func (s *JobServer) SaveJobLogs(ctx context.Context, req *pb.SaveJobLogsRequest) (*emptypb.Empty, error) {
	jobID, err := s.serviceCatalog.FetchModelID(ctx, req.JobId)
	if err != nil {
		return nil, err
	}

	if _, err = s.serviceCatalog.JobService.WriteLogs(ctx, jobID, int(req.StartOffset), []byte(req.Logs)); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ClaimJob claims the next available job for a runner.
func (s *JobServer) ClaimJob(ctx context.Context, req *pb.ClaimJobRequest) (*pb.ClaimJobResponse, error) {
	runnerID, err := s.serviceCatalog.FetchModelID(ctx, req.RunnerId)
	if err != nil {
		return nil, err
	}

	resp, err := s.serviceCatalog.JobService.ClaimJob(ctx, runnerID)
	if err != nil {
		return nil, err
	}

	return &pb.ClaimJobResponse{
		Job:   toPBJob(resp.Job),
		Token: resp.Token,
	}, nil
}

// SubscribeToJobLogStream subscribes to job log stream events.
func (s *JobServer) SubscribeToJobLogStream(req *pb.SubscribeToJobLogStreamRequest, stream pb.Jobs_SubscribeToJobLogStreamServer) error {
	jobID, err := s.serviceCatalog.FetchModelID(stream.Context(), req.JobId)
	if err != nil {
		return err
	}

	options := &job.LogStreamEventSubscriptionOptions{
		JobID: jobID,
	}

	if req.LastSeenLogSize != nil {
		lastSeenLogSize := int(*req.LastSeenLogSize)
		options.LastSeenLogSize = &lastSeenLogSize
	}

	eventChan, err := s.serviceCatalog.JobService.SubscribeToLogStreamEvents(stream.Context(), options)
	if err != nil {
		return err
	}

	for event := range eventChan {
		pbEvent := &pb.JobLogStreamEvent{
			Completed: event.Completed,
			Size:      int32(event.Size),
		}

		if event.Data != nil {
			pbEvent.Data = &pb.JobLogStreamEventData{
				Offset: int32(event.Data.Offset),
				Logs:   string(event.Data.Logs),
			}
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// SubscribeToJobEvents subscribes to job events.
func (s *JobServer) SubscribeToJobEvents(req *pb.SubscribeToJobEventsRequest, stream pb.Jobs_SubscribeToJobEventsServer) error {
	var runnerID, workspaceID *string

	if req.RunnerId != nil {
		id, err := s.serviceCatalog.FetchModelID(stream.Context(), *req.RunnerId)
		if err != nil {
			return err
		}
		runnerID = &id
	}

	if req.WorkspaceId != nil {
		id, err := s.serviceCatalog.FetchModelID(stream.Context(), *req.WorkspaceId)
		if err != nil {
			return err
		}
		workspaceID = &id
	}

	options := &job.SubscribeToJobsInput{
		RunnerID:    runnerID,
		WorkspaceID: workspaceID,
	}

	eventChan, err := s.serviceCatalog.JobService.SubscribeToJobs(stream.Context(), options)
	if err != nil {
		return err
	}

	for event := range eventChan {
		pbEvent := &pb.JobEvent{
			Action: event.Action,
			Job:    toPBJob(event.Job),
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// SubscribeToJobCancellationEvent subscribes to job cancellation events.
func (s *JobServer) SubscribeToJobCancellationEvent(req *pb.SubscribeToJobCancellationEventRequest, stream pb.Jobs_SubscribeToJobCancellationEventServer) error {
	jobID, err := s.serviceCatalog.FetchModelID(stream.Context(), req.JobId)
	if err != nil {
		return err
	}

	options := &job.CancellationSubscriptionsOptions{
		JobID: jobID,
	}

	eventChan, err := s.serviceCatalog.JobService.SubscribeToCancellationEvent(stream.Context(), options)
	if err != nil {
		return err
	}

	for event := range eventChan {
		pbEvent := &pb.JobCancellationEvent{
			Job: toPBJob(&event.Job),
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// toPBJob converts from Job model to ProtoBuf model.
func toPBJob(j *models.Job) *pb.Job {
	return &pb.Job{
		Metadata:        toPBMetadata(&j.Metadata, types.JobModelType),
		WorkspaceId:     gid.ToGlobalID(types.WorkspaceModelType, j.WorkspaceID),
		RunId:           gid.ToGlobalID(types.RunModelType, j.RunID),
		Type:            string(j.Type),
		Status:          pb.JobStatus(pb.JobStatus_value[string(j.GetStatus())]),
		MaxJobDuration:  j.MaxJobDuration,
		Properties:      j.Properties,
		CancelRequested: j.GetStatus() == models.JobCanceling,
		ForceCanceled:   j.ForceCanceled,
	}
}
