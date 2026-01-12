package tools

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

const (
	// defaultLogLimit is the default number of bytes to retrieve when fetching job logs (10 KiB)
	// Conservative limit to preserve LLM context window space for conversation
	defaultLogLimit = 10 * 1024
	// maxLogLimit is the maximum number of bytes that can be retrieved in a single request (50 KiB)
	// Conservative limit to avoid overwhelming LLM context windows
	maxLogLimit = 50 * 1024
)

// job represents a Terraform execution job (plan or apply).
type job struct {
	JobID                    string           `json:"job_id" jsonschema:"Unique identifier for this job"`
	TRN                      string           `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:job:group/workspace/job-id)"`
	Status                   models.JobStatus `json:"status" jsonschema:"Current status: queued (waiting), pending (assigned to runner), running (executing), or finished (completed)"`
	Type                     models.JobType   `json:"type" jsonschema:"Job type: plan (preview changes) or apply (execute changes)"`
	WorkspaceID              string           `json:"workspace_id" jsonschema:"ID of the workspace where this job is running"`
	RunID                    string           `json:"run_id" jsonschema:"ID of the parent run that contains this job"`
	RunnerID                 *string          `json:"runner_id,omitempty" jsonschema:"ID of the runner agent executing this job (null if not yet assigned)"`
	CancelRequested          bool             `json:"cancel_requested" jsonschema:"True if cancellation has been requested but not yet completed"`
	QueuedTimestamp          *time.Time       `json:"queued_timestamp,omitempty" jsonschema:"When the job entered the queue"`
	PendingTimestamp         *time.Time       `json:"pending_timestamp,omitempty" jsonschema:"When the job was assigned to a runner"`
	RunningTimestamp         *time.Time       `json:"running_timestamp,omitempty" jsonschema:"When the job started executing"`
	FinishedTimestamp        *time.Time       `json:"finished_timestamp,omitempty" jsonschema:"When the job completed"`
	CancelRequestedTimestamp *time.Time       `json:"cancel_requested_timestamp,omitempty" jsonschema:"When cancellation was requested"`
}

// getJobInput defines the parameters for retrieving a job.
type getJobInput struct {
	ID string `json:"id" jsonschema:"required,Job ID or TRN (e.g. Ul8yZ... or trn:job:workspace-path/job-id)"`
}

// getJobOutput wraps the job response.
type getJobOutput struct {
	Job job `json:"job" jsonschema:"The job execution details"`
}

// GetJob returns an MCP tool for retrieving job status and execution details.
// Use this to monitor job progress and check runner assignment.
func GetJob(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getJobInput, getJobOutput]) {
	tool := mcp.Tool{
		Name:        "get_job",
		Description: "Retrieve job status and execution timeline. A job is the actual execution of a plan or apply. Check this to see which runner is executing the job and track progress through queued, pending, running, and finished states.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Job",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getJobInput) (*mcp.CallToolResult, getJobOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getJobOutput{}, WrapMCPToolError(err, "failed to resolve job %q", input.ID)
		}

		j, ok := fetchedModel.(*models.Job)
		if !ok {
			return nil, getJobOutput{}, NewMCPToolError("job with id %s not found", input.ID)
		}

		output := getJobOutput{
			Job: job{
				JobID:                    j.GetGlobalID(),
				TRN:                      j.Metadata.TRN,
				Status:                   j.Status,
				Type:                     j.Type,
				WorkspaceID:              gid.ToGlobalID(types.WorkspaceModelType, j.WorkspaceID),
				RunID:                    gid.ToGlobalID(types.RunModelType, j.RunID),
				CancelRequested:          j.CancelRequestedTimestamp != nil,
				QueuedTimestamp:          j.Timestamps.QueuedTimestamp,
				PendingTimestamp:         j.Timestamps.PendingTimestamp,
				RunningTimestamp:         j.Timestamps.RunningTimestamp,
				FinishedTimestamp:        j.Timestamps.FinishedTimestamp,
				CancelRequestedTimestamp: j.CancelRequestedTimestamp,
			},
		}

		if j.RunnerID != nil {
			runnerGID := gid.ToGlobalID(types.RunnerModelType, *j.RunnerID)
			output.Job.RunnerID = &runnerGID
		}

		return nil, output, nil
	}

	return tool, handler
}

// getLatestJobInput defines the parameters for retrieving the latest job for a plan or apply.
type getLatestJobInput struct {
	ID string `json:"id" jsonschema:"required,Plan or Apply ID or TRN (e.g. Ul8yZ... or trn:plan:group/workspace/plan-id or trn:apply:group/workspace/apply-id)"`
}

// getLatestJobOutput wraps the job response.
type getLatestJobOutput struct {
	Job job `json:"job" jsonschema:"The latest job execution details"`
}

// GetLatestJob returns an MCP tool for retrieving the latest job for a plan or apply.
func GetLatestJob(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getLatestJobInput, getLatestJobOutput]) {
	tool := mcp.Tool{
		Name:        "get_latest_job",
		Description: "Get the latest job for a plan or apply. A plan or apply can have multiple jobs if retried. This returns the most recent one.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Latest Job",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getLatestJobInput) (*mcp.CallToolResult, getLatestJobOutput, error) {
		runService := tc.servicesCatalog.RunService

		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getLatestJobOutput{}, WrapMCPToolError(err, "failed to resolve %q", input.ID)
		}

		var j *models.Job
		switch m := fetchedModel.(type) {
		case *models.Plan:
			j, err = runService.GetLatestJobForPlan(ctx, m.Metadata.ID)
			if err != nil {
				return nil, getLatestJobOutput{}, WrapMCPToolError(err, "failed to get latest job for plan")
			}
		case *models.Apply:
			j, err = runService.GetLatestJobForApply(ctx, m.Metadata.ID)
			if err != nil {
				return nil, getLatestJobOutput{}, WrapMCPToolError(err, "failed to get latest job for apply")
			}
		default:
			return nil, getLatestJobOutput{}, NewMCPToolError("expected plan or apply ID, got %T", fetchedModel)
		}

		output := getLatestJobOutput{
			Job: job{
				JobID:                    j.GetGlobalID(),
				TRN:                      j.Metadata.TRN,
				Status:                   j.Status,
				Type:                     j.Type,
				WorkspaceID:              gid.ToGlobalID(types.WorkspaceModelType, j.WorkspaceID),
				RunID:                    gid.ToGlobalID(types.RunModelType, j.RunID),
				CancelRequested:          j.CancelRequestedTimestamp != nil,
				QueuedTimestamp:          j.Timestamps.QueuedTimestamp,
				PendingTimestamp:         j.Timestamps.PendingTimestamp,
				RunningTimestamp:         j.Timestamps.RunningTimestamp,
				FinishedTimestamp:        j.Timestamps.FinishedTimestamp,
				CancelRequestedTimestamp: j.CancelRequestedTimestamp,
			},
		}

		if j.RunnerID != nil {
			runnerGID := gid.ToGlobalID(types.RunnerModelType, *j.RunnerID)
			output.Job.RunnerID = &runnerGID
		}

		return nil, output, nil
	}

	return tool, handler
}

// getJobLogsInput defines the parameters for retrieving job logs.
type getJobLogsInput struct {
	ID    string `json:"id" jsonschema:"required,Job ID or TRN (e.g. Ul8yZ... or trn:job:workspace-path/job-id)"`
	Start *int   `json:"start,omitempty" jsonschema:"Byte offset to start reading from (default: 0, use for pagination)"`
	Limit *int   `json:"limit,omitempty" jsonschema:"Maximum bytes to return (default: 10000, max: 50000)"`
}

// getJobLogsOutput contains the retrieved log content and pagination info.
type getJobLogsOutput struct {
	JobID   string `json:"job_id" jsonschema:"The job ID these logs belong to"`
	Logs    string `json:"logs" jsonschema:"The log content as text"`
	Start   int    `json:"start" jsonschema:"The byte offset where these logs start"`
	Size    int    `json:"size" jsonschema:"Number of bytes returned in this response"`
	HasMore bool   `json:"has_more" jsonschema:"True if more logs are available (use start + size for next request)"`
}

// GetJobLogs returns an MCP tool for retrieving job logs with pagination.
// Logs are returned in chunks to avoid overwhelming context windows.
func GetJobLogs(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getJobLogsInput, getJobLogsOutput]) {
	tool := mcp.Tool{
		Name:        "get_job_logs",
		Description: "Retrieve Terraform execution logs from a job. Logs are paginated - check has_more and use start parameter to fetch additional chunks.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Job Logs",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getJobLogsInput) (*mcp.CallToolResult, getJobLogsOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getJobLogsOutput{}, WrapMCPToolError(err, "failed to resolve job %q", input.ID)
		}

		j, ok := fetchedModel.(*models.Job)
		if !ok {
			return nil, getJobLogsOutput{}, NewMCPToolError("job with id %s not found", input.ID)
		}

		start := 0
		if input.Start != nil {
			start = *input.Start
		}

		limit := defaultLogLimit
		if input.Limit != nil {
			if *input.Limit > maxLogLimit {
				return nil, getJobLogsOutput{}, NewMCPToolError("limit %d exceeds maximum allowed limit of %d bytes", *input.Limit, maxLogLimit)
			}
			limit = *input.Limit
		}

		// Request one extra byte to detect if there's more data
		logs, err := tc.servicesCatalog.JobService.ReadLogs(ctx, j.Metadata.ID, start, limit+1)
		if err != nil {
			return nil, getJobLogsOutput{}, WrapMCPToolError(err, "failed to get logs for job")
		}

		// If we got more than limit, there's more data available
		hasMore := len(logs) > limit
		if hasMore {
			logs = logs[:limit]
		}

		return nil, getJobLogsOutput{
			JobID:   j.GetGlobalID(),
			Logs:    string(logs),
			Start:   start,
			Size:    len(logs),
			HasMore: hasMore,
		}, nil
	}

	return tool, handler
}
