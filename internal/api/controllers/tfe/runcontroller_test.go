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
	"github.com/hashicorp/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
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

func TestCanReadRunLogs(t *testing.T) {
	jobCaller := func(runID string, status models.JobStatus) *auth.JobCaller {
		job := &models.Job{}
		require.NoError(t, job.SetStatus(status))
		return &auth.JobCaller{RunID: runID, Job: job}
	}

	run := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}

	tests := []struct {
		name   string
		caller auth.Caller
		expect bool
	}{
		{name: "job for the same run", caller: jobCaller("run-1", models.JobRunning), expect: true},
		// A job can view runs across its root group, but must not be handed another run's logs.
		{name: "job for a different run", caller: jobCaller("run-2", models.JobRunning), expect: false},
		{name: "finished job for the same run", caller: jobCaller("run-1", models.JobFinished), expect: false},
		{name: "non-job caller", caller: auth.NewMockCaller(t), expect: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, canReadRunLogs(auth.WithCaller(context.Background(), test.caller), run))
		})
	}
}

// newGoTFEClient serves the given controllers' routes and returns a go-tfe client pointed at them, so a
// test exercises the requests go-tfe (and so the Terraform CLI) actually sends.
func newGoTFEClient(t *testing.T, routes ...interface{ RegisterRoutes(chi.Router) }) *gotfe.Client {
	router := chi.NewRouter()
	for _, r := range routes {
		r.RegisterRoutes(router)
	}
	apiRouter := chi.NewRouter()
	apiRouter.Mount(gotfe.DefaultBasePath, router)

	server := httptest.NewServer(apiRouter)
	t.Cleanup(server.Close)

	client, err := gotfe.NewClient(&gotfe.Config{Address: server.URL, Token: "token", RetryServerErrors: false})
	require.NoError(t, err)
	return client
}

// TestRunController_RunActionComments checks the comment go-tfe sends with apply and cancel reaches the
// service. go-tfe encodes these options as plain JSON rather than JSON:API.
func TestRunController_RunActionComments(t *testing.T) {
	const runID = "e5f6a7b8-1c39-4c8f-9a2f-2c4d5e6f7a80"

	tests := []struct {
		name    string
		comment *string
	}{
		{name: "with a comment", comment: ptr.String("ship it")},
		{name: "without a comment"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := &models.Run{Metadata: models.ResourceMetadata{ID: runID}}
			mockRunService := runservice.NewMockService(t)
			mockRunService.On("ApplyRun", mock.Anything, runID, test.comment).Return(result, nil)
			mockRunService.On("CancelRun", mock.Anything, &runservice.CancelRunInput{RunID: runID, Comment: test.comment}).
				Return(result, nil)

			testLogger, _ := logger.NewForTest()
			client := newGoTFEClient(t, &runController{
				respWriter:        response.NewWriter(testLogger),
				jwtAuthMiddleware: func(next http.Handler) http.Handler { return next },
				logger:            testLogger,
				runService:        mockRunService,
			})

			runGID := result.GetGlobalID()
			require.NoError(t, client.Runs.Apply(context.Background(), runGID, gotfe.RunApplyOptions{Comment: test.comment}))
			require.NoError(t, client.Runs.Cancel(context.Background(), runGID, gotfe.RunCancelOptions{Comment: test.comment}))
		})
	}
}

// TestRunController_RunActionBodies covers the bodies apply and cancel accept besides go-tfe's: an empty
// body is no options, and a body that is not a JSON object is the client's error.
func TestRunController_RunActionBodies(t *testing.T) {
	const runID = "e5f6a7b8-1c39-4c8f-9a2f-2c4d5e6f7a80"

	tests := []struct {
		name          string
		body          string
		expectCode    int
		expectComment *string
	}{
		{name: "empty body", body: "", expectCode: http.StatusOK},
		{name: "empty object", body: "{}", expectCode: http.StatusOK},
		{name: "comment", body: `{"comment":"why"}`, expectCode: http.StatusOK, expectComment: ptr.String("why")},
		{name: "JSON:API document is not the options", body: `{"data":{"type":"runs"}}`, expectCode: http.StatusOK},
		{name: "not JSON", body: "comment=why", expectCode: http.StatusBadRequest},
		{name: "wrong comment type", body: `{"comment":1}`, expectCode: http.StatusBadRequest},
	}

	for _, test := range tests {
		for _, action := range []string{"apply", "cancel"} {
			t.Run(action+": "+test.name, func(t *testing.T) {
				result := &models.Run{Metadata: models.ResourceMetadata{ID: runID}}
				mockRunService := runservice.NewMockService(t)
				if test.expectCode == http.StatusOK {
					if action == "apply" {
						mockRunService.On("ApplyRun", mock.Anything, runID, test.expectComment).Return(result, nil)
					} else {
						mockRunService.On("CancelRun", mock.Anything, &runservice.CancelRunInput{RunID: runID, Comment: test.expectComment}).
							Return(result, nil)
					}
				}

				testLogger, _ := logger.NewForTest()
				c := &runController{respWriter: response.NewWriter(testLogger), logger: testLogger, runService: mockRunService}
				handler := c.ApplyRun
				if action == "cancel" {
					handler = c.CancelRun
				}

				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", result.GetGlobalID())
				req := httptest.NewRequest(http.MethodPost, "/runs/x/actions/"+action, bytes.NewBufferString(test.body))
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

				rec := httptest.NewRecorder()
				handler(rec, req)
				assert.Equal(t, test.expectCode, rec.Code, rec.Body.String())
			})
		}
	}
}

// TestRunController_CreateRunPlanOnlyAndVersion checks go-tfe's plan-only and terraform-version run
// attributes reach run creation as a speculative run and its Terraform version.
func TestRunController_CreateRunPlanOnlyAndVersion(t *testing.T) {
	const (
		workspaceID = "a1b2c3d4-1c39-4c8f-9a2f-2c4d5e6f7a80"
		runID       = "e5f6a7b8-1c39-4c8f-9a2f-2c4d5e6f7a80"
	)

	tests := []struct {
		name              string
		planOnly          *bool
		terraformVersion  *string
		expectSpeculative *bool
		expectVersion     string
	}{
		{name: "neither sent"},
		{
			name:              "plan-only with a version",
			planOnly:          ptr.Bool(true),
			terraformVersion:  ptr.String("1.9.0"),
			expectSpeculative: ptr.Bool(true),
			expectVersion:     "1.9.0",
		},
		{
			// An explicit false leaves the configuration version to decide, as an omitted one does.
			name:     "plan-only false",
			planOnly: ptr.Bool(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockRunService := runservice.NewMockService(t)
			mockRunService.On("CreateRun", mock.Anything, mock.MatchedBy(func(input *runservice.CreateRunInput) bool {
				return reflect.DeepEqual(test.expectSpeculative, input.Speculative) &&
					input.TerraformVersion == test.expectVersion && input.WorkspaceID == workspaceID
			})).Return(&models.Run{Metadata: models.ResourceMetadata{ID: runID}, WorkspaceID: workspaceID}, nil)

			testLogger, _ := logger.NewForTest()
			client := newGoTFEClient(t, &runController{
				respWriter:        response.NewWriter(testLogger),
				jwtAuthMiddleware: func(next http.Handler) http.Handler { return next },
				logger:            testLogger,
				runService:        mockRunService,
			})

			_, err := client.Runs.Create(context.Background(), gotfe.RunCreateOptions{
				Workspace:        &gotfe.Workspace{ID: (&models.Workspace{Metadata: models.ResourceMetadata{ID: workspaceID}}).GetGlobalID()},
				PlanOnly:         test.planOnly,
				TerraformVersion: test.terraformVersion,
			})
			require.NoError(t, err)
		})
	}
}

func FuzzParseRunVariables(f *testing.F) {
	f.Add([]byte(`{"data":{"type":"runs","attributes":{"variables":[{"key":"a","value":"1"}]}}}`))
	f.Add([]byte(`{"data":{"type":"runs","attributes":{"variables":[{"Key":"a","Value":"1"}]}}}`))
	f.Add([]byte(`{"data":{"type":"runs","attributes":{"variables":[null]}}}`))
	f.Add([]byte(`{"data":{"type":"runs","attributes":{"variables":[{"key":""},{"Key":"b"}]}}}`))
	f.Add([]byte(`{"data":{"type":"runs","attributes":{"variables":[{"Key":1}]}}}`))
	f.Add([]byte(`{"data":{"type":"runs","attributes":{"variables":[{"key":"a","Key":"b","value":"x","Value":"y"}]}}}`))
	f.Add([]byte(`{"data":{"type":"runs","attributes":{"variables":[]}}}`))

	f.Fuzz(func(t *testing.T, body []byte) {
		var req gotfe.RunCreateOptions
		if err := jsonapi.UnmarshalPayload(bytes.NewReader(body), &req); err != nil {
			t.Skip()
		}

		variables, err := parseRunVariables(req, body)
		if err != nil {
			assert.Nil(t, variables)
			return
		}

		var keyed []string
		for _, v := range req.Variables {
			if v != nil && v.Key != "" {
				keyed = append(keyed, v.Key)
			}
		}

		for _, v := range variables {
			assert.Equal(t, models.TerraformVariableCategory, v.Category)
			require.NotNil(t, v.Value)
		}

		// Variables in the current shape are used as sent, and the fallback only applies when none
		// of them had a key.
		if len(keyed) > 0 {
			require.Len(t, variables, len(keyed))
			for i, key := range keyed {
				assert.Equal(t, key, variables[i].Key)
			}
		} else if len(req.Variables) == 0 {
			assert.Empty(t, variables)
		}
	})
}
