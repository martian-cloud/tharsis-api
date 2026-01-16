// Package servers implements the gRPC servers.
package servers

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
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
		Metadata:       toPBMetadata(&j.Metadata, types.JobModelType),
		WorkspaceId:    gid.ToGlobalID(types.WorkspaceModelType, j.WorkspaceID),
		RunId:          gid.ToGlobalID(types.RunModelType, j.RunID),
		Type:           string(j.Type),
		Status:         string(j.Status),
		MaxJobDuration: j.MaxJobDuration,
	}
}
