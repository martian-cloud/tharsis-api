// Package runtest provides fake Tharsis run, run gate and job clients, and run snapshot builders,
// for testing code that follows runs.
package runtest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Stream is a gRPC client stream whose messages come from recv.
type Stream[T any] struct {
	grpc.ClientStream
	recv func() (*T, error)
}

// Recv returns the next message.
func (s *Stream[T]) Recv() (*T, error) { return s.recv() }

// Runs serves run snapshots from a queue: each GetRunByID returns the head, popping it while more
// remain, so a test scripts the run's progress. Actions can replace the queue. It also serves the
// run gates client, since approving a gate changes what the run's policy check results report.
// Fields may be set before following starts, and read after it ends; hooks run under its lock.
type Runs struct {
	pb.RunsClient
	pb.RunGatesClient

	mu          sync.Mutex
	Queue       []*pb.Run
	Results     map[string]*pb.PolicyCheckResults
	ApproveErr  error
	OverrideErr error
	DiscardErr  error
	ApplyErr    error
	CancelErr   error
	OnApply     func(*Runs)
	OnDiscard   func(*Runs)
	OnApprove   func(*Runs) *pb.RunGate
	OnOverride  func(*Runs) *pb.RunGate
	// HoldUntil, when set, keeps serving the head of the queue until it reports true, so a test can
	// tie the run's progress to what the follower has done rather than to how often it reads the run.
	HoldUntil func() bool
	// GetErrs are returned, in order, by the next GetRunByID calls before snapshots are served again.
	GetErrs []error

	Applied, Discarded, Approved, Overridden, Canceled, Gets int
}

// NewRuns returns a Runs serving the given snapshots.
func NewRuns(runs ...*pb.Run) *Runs {
	return &Runs{Queue: runs, Results: map[string]*pb.PolicyCheckResults{}}
}

// Set replaces the queue. Call it from a hook, which runs under the lock.
func (f *Runs) Set(runs ...*pb.Run) {
	f.Queue = runs
}

// Count reads a counter under the lock.
func (f *Runs) Count(counter *int) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *counter
}

// GetRunByID serves the head of the queue.
func (f *Runs) GetRunByID(context.Context, *pb.GetRunByIDRequest, ...grpc.CallOption) (*pb.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Gets++
	if len(f.GetErrs) > 0 {
		err := f.GetErrs[0]
		f.GetErrs = f.GetErrs[1:]
		return nil, err
	}
	r := f.Queue[0]
	if len(f.Queue) > 1 && (f.HoldUntil == nil || f.HoldUntil()) {
		f.Queue = f.Queue[1:]
	}
	return proto.Clone(r).(*pb.Run), nil
}

// SubscribeToRunEvents emits an event every few milliseconds until the subscription is torn down.
func (f *Runs) SubscribeToRunEvents(ctx context.Context, _ *pb.SubscribeToRunEventsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.RunEvent], error) {
	return &Stream[pb.RunEvent]{recv: func() (*pb.RunEvent, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Millisecond):
			return &pb.RunEvent{}, nil
		}
	}}, nil
}

// ApplyRun records an apply.
func (f *Runs) ApplyRun(context.Context, *pb.ApplyRunRequest, ...grpc.CallOption) (*pb.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Applied++
	if f.OnApply != nil {
		f.OnApply(f)
	}
	if f.ApplyErr != nil {
		return nil, f.ApplyErr
	}
	return &pb.Run{}, nil
}

// DiscardRun records a discard, unless DiscardErr is set.
func (f *Runs) DiscardRun(context.Context, *pb.DiscardRunRequest, ...grpc.CallOption) (*pb.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.OnDiscard != nil {
		f.OnDiscard(f)
	}
	if f.DiscardErr != nil {
		return nil, f.DiscardErr
	}
	f.Discarded++
	return &pb.Run{}, nil
}

// CancelRun records a cancel, unless CancelErr is set.
func (f *Runs) CancelRun(context.Context, *pb.CancelRunRequest, ...grpc.CallOption) (*pb.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.CancelErr != nil {
		return nil, f.CancelErr
	}
	f.Canceled++
	return &pb.Run{}, nil
}

// ApproveRunGate records an approval and returns what OnApprove does.
func (f *Runs) ApproveRunGate(_ context.Context, req *pb.ApproveRunGateRequest, _ ...grpc.CallOption) (*pb.RunGate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if req.Decision != pb.RunGateDecision_RUN_GATE_DECISION_APPROVE {
		return nil, fmt.Errorf("unexpected run gate decision %s", req.Decision)
	}
	if f.ApproveErr != nil {
		return nil, f.ApproveErr
	}
	f.Approved++
	return f.OnApprove(f), nil
}

// OverrideRunGate records an override and returns what OnOverride does.
func (f *Runs) OverrideRunGate(context.Context, *pb.OverrideRunGateRequest, ...grpc.CallOption) (*pb.RunGate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.OverrideErr != nil {
		return nil, f.OverrideErr
	}
	f.Overridden++
	return f.OnOverride(f), nil
}

// GetPolicyCheckResults serves Results.
func (f *Runs) GetPolicyCheckResults(_ context.Context, req *pb.GetPolicyCheckResultsRequest, _ ...grpc.CallOption) (*pb.PolicyCheckResults, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.Results[req.PolicyCheckId]
	if !ok {
		return nil, status.Error(codes.NotFound, "no results")
	}
	return proto.Clone(r).(*pb.PolicyCheckResults), nil
}

// Jobs streams one log chunk per job and records the order jobs were streamed in.
type Jobs struct {
	pb.JobsClient

	mu       sync.Mutex
	streamed []string
}

// SubscribeToJobLogStream streams "logs for <job ID>" and completes.
func (f *Jobs) SubscribeToJobLogStream(_ context.Context, req *pb.SubscribeToJobLogStreamRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pb.JobLogStreamEvent], error) {
	f.mu.Lock()
	f.streamed = append(f.streamed, req.JobId)
	f.mu.Unlock()
	logs := "logs for " + req.JobId
	return &Stream[pb.JobLogStreamEvent]{recv: func() (*pb.JobLogStreamEvent, error) {
		return &pb.JobLogStreamEvent{Completed: true, Size: int32(len(logs)), Data: &pb.JobLogStreamEventData{Logs: logs}}, nil
	}}, nil
}

// Streamed returns the jobs streamed, in order.
func (f *Jobs) Streamed() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.streamed...)
}

// Client returns a gRPC client backed by the fakes.
func Client(runs *Runs, jobs *Jobs) *client.GRPCClient {
	return &client.GRPCClient{RunsClient: runs, RunGatesClient: runs, JobsClient: jobs}
}

// Run snapshot builders.

// RunOpt changes a run snapshot.
type RunOpt func(*pb.Run)

// Task stage names.
const (
	PrePlan   = pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN
	PostPlan  = pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_PLAN
	PreApply  = pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_APPLY
	PostApply = pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY
)

// NewRun returns a snapshot of run-1 in the given status, with a created plan and apply.
func NewRun(runStatus pb.RunStatus, opts ...RunOpt) *pb.Run {
	r := &pb.Run{
		Metadata:    &pb.ResourceMetadata{Id: "run-1"},
		WorkspaceId: "ws-1",
		Status:      runStatus,
		Plan:        &pb.Plan{Status: pb.PlanStatus_CREATED},
		Apply:       &pb.Apply{Status: pb.ApplyStatus_CREATED},
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// WithAutoApply marks the run auto-apply.
func WithAutoApply() RunOpt {
	return func(r *pb.Run) { r.AutoApply = true }
}

// WithDestroy marks the run a destroy run.
func WithDestroy() RunOpt {
	return func(r *pb.Run) { r.IsDestroy = true }
}

// WithPlan sets the plan's status and latest job.
func WithPlan(s pb.PlanStatus, jobID string) RunOpt {
	return func(r *pb.Run) {
		r.Plan = &pb.Plan{Status: s}
		if jobID != "" {
			r.Plan.LatestJobId = ptr.String(jobID)
		}
	}
}

// WithApply sets the apply's status and latest job.
func WithApply(s pb.ApplyStatus, jobID string) RunOpt {
	return func(r *pb.Run) {
		r.Apply = &pb.Apply{Status: s}
		if jobID != "" {
			r.Apply.LatestJobId = ptr.String(jobID)
		}
	}
}

// Speculative makes the run speculative, with no apply.
func Speculative() RunOpt {
	return func(r *pb.Run) {
		r.Speculative = true
		r.Apply = nil
	}
}

// WithStage adds a task stage.
func WithStage(name pb.RunTaskStageName, s pb.RunTaskStageStatus, checks ...*pb.PolicyCheck) RunOpt {
	return func(r *pb.Run) {
		r.TaskStages = append(r.TaskStages, &pb.RunTaskStage{Id: "stage-" + name.String(), StageName: name, Status: s, PolicyChecks: checks})
	}
}

// NewCheck returns an OPA policy check.
func NewCheck(id string, s pb.PolicyCheckStatus, jobID string) *pb.PolicyCheck {
	c := &pb.PolicyCheck{Id: id, Status: s, CheckType: pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA}
	if jobID != "" {
		c.LatestJobId = ptr.String(jobID)
	}
	return c
}

// SoftFailedResults is a check whose one soft-mandatory policy failed, blocked on a pending gate.
func SoftFailedResults(checkID string, required, approvals int32) *pb.PolicyCheckResults {
	return &pb.PolicyCheckResults{
		PolicyCheck: &pb.PolicyCheck{
			Id:        checkID,
			CheckType: pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA,
			Status:    pb.PolicyCheckStatus_POLICY_CHECK_STATUS_SOFT_FAILED,
		},
		PolicyResults: []*pb.PolicyCheckPolicyResult{{
			Policy: &pb.PolicyCheckPolicy{
				Id:               "p1",
				EnforcementLevel: pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_SOFT_MANDATORY,
				Status:           pb.PolicyCheckPolicyStatus_POLICY_CHECK_POLICY_STATUS_FAILED,
				Provenance:       &pb.PolicyCheckPolicyProvenance{PolicyTrn: "trn:policy:group/encrypted-buckets"},
			},
			Messages: []string{"bucket must be encrypted"},
		}},
		RunGate: &pb.RunGate{
			Metadata:      &pb.ResourceMetadata{Id: "gate-1"},
			PolicyCheckId: checkID,
			Status:        pb.RunGateStatus_RUN_GATE_STATUS_PENDING,
			ApprovalRules: []*pb.RunGateApprovalRule{{Name: "p1", RequiredApprovals: required, Approvals: approvals}},
		},
	}
}

// AppliedRun is an applied run whose plan and apply finished.
func AppliedRun(opts ...RunOpt) *pb.Run {
	return NewRun(pb.RunStatus_APPLIED, append([]RunOpt{
		WithPlan(pb.PlanStatus_FINISHED, "plan-job"),
		WithApply(pb.ApplyStatus_FINISHED, "apply-job"),
	}, opts...)...)
}
