package tfe

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	jobservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	runservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func Test_parseRunVariables(t *testing.T) {
	type args struct {
		req  gotfe.RunCreateOptions
		body []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []runvariables.Variable
		wantErr bool
	}{
		{
			name: "No Variables should be fine",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{},
				},
				body: []byte{},
			},
			want:    []runvariables.Variable{},
			wantErr: false,
		},
		{
			name: "Proper API Variables are supported",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{
						{
							Key:   "foo",
							Value: "\"bar\"",
						},
					},
				},
				body: []byte{},
			},
			want: []runvariables.Variable{
				{
					Key:      "foo",
					Value:    ptr.String("\"bar\""),
					Category: models.TerraformVariableCategory,
				},
			},
			wantErr: false,
		},
		{
			name: "Broken Terraform API Variables are supported",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{
						nil,
						{
							Key:   "",
							Value: "",
						},
					},
				},
				body: []byte(`{"data":{"attributes":{"variables":[{"Key":"foo","Value":"\"bar\""}]}}}`),
			},
			want: []runvariables.Variable{
				{
					Key:      "foo",
					Value:    ptr.String("\"bar\""),
					Category: models.TerraformVariableCategory,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid JSON should result in an error",
			args: args{
				req: gotfe.RunCreateOptions{
					Variables: []*gotfe.RunVariable{
						nil,
						{
							Key:   "",
							Value: "",
						},
					},
				},
				body: []byte(`{"data":{"attributes":{"variables":[{"Key":"foo","Value":"\"bar\""}]}}`),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRunVariables(tt.args.req, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRunVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRunVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toTFEApplyStatus(t *testing.T) {
	tests := []struct {
		name   string
		status models.ApplyStatus
		want   gotfe.ApplyStatus
	}{
		{
			name:   "skipped maps to created since the CLI has no skipped status",
			status: models.ApplySkipped,
			want:   gotfe.ApplyStatus(models.ApplyCreated),
		},
		{
			name:   "created passes through",
			status: models.ApplyCreated,
			want:   gotfe.ApplyStatus(models.ApplyCreated),
		},
		{
			name:   "pending passes through",
			status: models.ApplyPending,
			want:   gotfe.ApplyStatus(models.ApplyPending),
		},
		{
			name:   "queued passes through",
			status: models.ApplyQueued,
			want:   gotfe.ApplyStatus(models.ApplyQueued),
		},
		{
			name:   "running passes through",
			status: models.ApplyRunning,
			want:   gotfe.ApplyStatus(models.ApplyRunning),
		},
		{
			name:   "finished passes through",
			status: models.ApplyFinished,
			want:   gotfe.ApplyStatus(models.ApplyFinished),
		},
		{
			name:   "errored passes through",
			status: models.ApplyErrored,
			want:   gotfe.ApplyStatus(models.ApplyErrored),
		},
		{
			name:   "canceled passes through",
			status: models.ApplyCanceled,
			want:   gotfe.ApplyStatus(models.ApplyCanceled),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toTFEApplyStatus(tt.status); got != tt.want {
				t.Errorf("toTFEApplyStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestGetPolicyCheckLogs(t *testing.T) {
	const (
		runID   = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
		checkID = "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
		jobID   = "c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
		wsID    = "d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14"
	)
	checkGID := models.RunNodeGID(checkID)
	logData := []byte("policy check output")

	tests := []struct {
		name           string
		getRunErr      error
		includeCheck   bool
		latestJobID    *string
		expectReadLogs bool
		wantStatus     int
		wantBody       string
	}{
		{
			name:           "finished check streams its job logs",
			includeCheck:   true,
			latestJobID:    ptr.String(jobID),
			expectReadLogs: true,
			wantStatus:     http.StatusOK,
			wantBody:       string(logData),
		},
		{
			name:         "check without a job yet returns an empty 200",
			includeCheck: true,
			latestJobID:  nil,
			wantStatus:   http.StatusOK,
			wantBody:     "",
		},
		{
			name:         "unknown policy check id returns 404",
			includeCheck: false,
			wantStatus:   http.StatusNotFound,
		},
		{
			name:       "error resolving the run is surfaced",
			getRunErr:  errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunService := runservice.NewMockService(t)
			mockJobService := jobservice.NewMockService(t)

			if tt.getRunErr != nil {
				mockRunService.On("GetRunByNodeID", mock.Anything, checkID).Return(nil, tt.getRunErr)
			} else {
				run := &models.Run{
					Metadata:    models.ResourceMetadata{ID: runID},
					WorkspaceID: wsID,
				}
				if tt.includeCheck {
					run.TaskStages = []*models.RunTaskStage{
						{StageName: models.RunTaskStageNamePostPlan, PolicyChecks: []*models.PolicyCheck{
							{ID: checkID, LatestJobID: tt.latestJobID},
						}},
					}
				}
				mockRunService.On("GetRunByNodeID", mock.Anything, checkID).Return(run, nil)
			}

			if tt.expectReadLogs {
				mockJobService.On("ReadLogs", mock.Anything, jobID, 0, policyCheckLogReadChunkSize).
					Return(io.NopCloser(bytes.NewReader(logData)), nil)
			}

			testLogger, _ := logger.NewForTest()
			c := &runController{
				respWriter: response.NewWriter(testLogger),
				logger:     testLogger,
				runService: mockRunService,
				jobService: mockJobService,
			}

			rec := httptest.NewRecorder()
			c.GetPolicyCheckLogs(rec, newPolicyCheckLogsRequest(checkGID))

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, tt.wantBody, rec.Body.String())
			}
		})
	}

	// Sanity check that the run node global ID round-trips back to the raw id the handler
	// resolves it to.
	assert.Equal(t, checkID, gid.FromGlobalID(checkGID))
}

// TestGetPolicyCheckLogs_StreamsAllChunks verifies the handler reads the job's logs in
// successive <=1 MiB windows and concatenates them, stopping once a short (final) chunk
// arrives. go-tfe fetches this output in a single unpaginated request, so the endpoint must
// return the complete log even when it spans more than one read window.
func TestGetPolicyCheckLogs_StreamsAllChunks(t *testing.T) {
	const (
		checkID = "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
		jobID   = "c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
	)
	checkGID := models.RunNodeGID(checkID)

	firstChunk := bytes.Repeat([]byte("a"), policyCheckLogReadChunkSize)
	finalChunk := []byte("tail")

	mockRunService := runservice.NewMockService(t)
	mockJobService := jobservice.NewMockService(t)

	mockRunService.On("GetRunByNodeID", mock.Anything, checkID).Return(&models.Run{
		TaskStages: []*models.RunTaskStage{
			{StageName: models.RunTaskStageNamePostPlan, PolicyChecks: []*models.PolicyCheck{{ID: checkID, LatestJobID: ptr.String(jobID)}}},
		},
	}, nil)

	// A full first window signals there may be more, so the handler advances the offset and
	// reads again; the short second window ends the stream.
	mockJobService.On("ReadLogs", mock.Anything, jobID, 0, policyCheckLogReadChunkSize).
		Return(io.NopCloser(bytes.NewReader(firstChunk)), nil).Once()
	mockJobService.On("ReadLogs", mock.Anything, jobID, policyCheckLogReadChunkSize, policyCheckLogReadChunkSize).
		Return(io.NopCloser(bytes.NewReader(finalChunk)), nil).Once()

	testLogger, _ := logger.NewForTest()
	c := &runController{
		respWriter: response.NewWriter(testLogger),
		logger:     testLogger,
		runService: mockRunService,
		jobService: mockJobService,
	}

	rec := httptest.NewRecorder()
	c.GetPolicyCheckLogs(rec, newPolicyCheckLogsRequest(checkGID))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, len(firstChunk)+len(finalChunk), rec.Body.Len())
	assert.True(t, bytes.HasSuffix(rec.Body.Bytes(), finalChunk))
}

// newPolicyCheckLogsRequest builds a GET request whose chi route context carries the policy
// check id, so GetPolicyCheckLogs can be invoked directly without mounting the router (and its
// JWT middleware).
func newPolicyCheckLogsRequest(checkGID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", checkGID)
	req := httptest.NewRequest(http.MethodGet, "/policy-checks/"+checkGID+"/output", nil)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
