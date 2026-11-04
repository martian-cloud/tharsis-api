package tfe

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	runservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	workspaceservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TestWorkspaceController_CreateStateVersion covers the optional json-state attribute a Terraform CLI
// state push sends alongside the state itself. The controller only forwards it; decoding and storage are
// the service's job, which is what keeps the rendering in the same transaction as the state version.
func TestWorkspaceController_CreateStateVersion(t *testing.T) {
	const (
		workspaceID    = "a1b2c3d4-1c39-4c8f-9a2f-2c4d5e6f7a80"
		stateVersionID = "b8e2f2b4-1c39-4c8f-9a2f-2c4d5e6f7a80"
		apiURL         = "https://tharsis.example.com"
		versionedPath  = "/tfe/v2"
	)

	rawState := base64.StdEncoding.EncodeToString([]byte(`{"version":4,"serial":1}`))
	jsonState := base64.StdEncoding.EncodeToString([]byte(`{"format_version":"1.0"}`))
	objectStoreKey := "workspaces/" + workspaceID + "/state_versions/" + stateVersionID + ".json"

	tests := []struct {
		name string
		// jsonState is the base64 json-state attribute, or empty to omit it.
		jsonState string
		// createError is returned by the service instead of a state version.
		createError error
		expectCode  int
		// expectJSONDownloadURL is whether the response advertises the rendering.
		expectJSONDownloadURL bool
	}{
		{
			// The common case today: an older CLI, or one that produced no rendering.
			name:       "no json-state is forwarded as nil",
			expectCode: http.StatusCreated,
		},
		{
			// The service returns a state version whose key is already set, so unlike the previous
			// create-then-upload sequence the response can advertise the rendering immediately.
			name:                  "json-state is forwarded and advertised in the response",
			jsonState:             jsonState,
			expectCode:            http.StatusCreated,
			expectJSONDownloadURL: true,
		},
		{
			// Decoding now happens in the service, so a malformed rendering arrives here as an EInvalid.
			name:        "a rejected rendering surfaces the service status",
			jsonState:   "not-base64!!",
			createError: errors.New("failed to decode base64-encoded state version JSON", errors.WithErrorCode(errors.EInvalid)),
			expectCode:  http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaceService := workspaceservice.NewMockService(t)

			// The fourth argument is the assertion that matters: the base64 attribute is handed to the
			// service untouched, or nil when the client sent none.
			var expectJSONArg any
			if test.jsonState == "" {
				expectJSONArg = mock.MatchedBy(func(arg *string) bool { return arg == nil })
			} else {
				expectJSONArg = mock.MatchedBy(func(arg *string) bool { return arg != nil && *arg == test.jsonState })
			}

			created := &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID:                   stateVersionID,
					CreationTimestamp:    ptr.Time(time.Now().UTC()),
					LastUpdatedTimestamp: ptr.Time(time.Now().UTC()),
				},
				WorkspaceID: workspaceID,
			}
			if test.expectJSONDownloadURL {
				created.JSONObjectStoreKey = ptr.String(objectStoreKey)
			}
			if test.createError != nil {
				created = nil
			}

			mockWorkspaceService.On("CreateStateVersion", mock.Anything, mock.Anything, rawState, expectJSONArg).
				Return(created, test.createError)

			testLogger, _ := logger.NewForTest()
			c := &workspaceController{
				respWriter:       response.NewWriter(testLogger),
				logger:           testLogger,
				workspaceService: mockWorkspaceService,
				tharsisAPIURL:    apiURL,
				tfeVersionedPath: versionedPath,

				stateVersionMaxUploadSize: 1024 * 1024,
			}

			rec := httptest.NewRecorder()
			c.CreateStateVersion(rec, newCreateStateVersionRequest(t,
				gid.ToGlobalID(types.WorkspaceModelType, workspaceID), rawState, test.jsonState))

			assert.Equal(t, test.expectCode, rec.Code)
			if test.createError == nil {
				// go-tfe's attributes carry no omitempty, so the field is always present; its value is
				// what says whether a rendering is downloadable. The upload URL is always advertised, so
				// matching on the key rather than on "/content.json" keeps the two apart.
				expectValue := ""
				if test.expectJSONDownloadURL {
					expectValue = apiURL + versionedPath + "/state-versions/" +
						gid.ToGlobalID(types.StateVersionModelType, stateVersionID) + "/content.json"
				}
				assert.Contains(t, rec.Body.String(),
					`"hosted-json-state-download-url":"`+expectValue+`"`)
			}
		})
	}
}

// newCreateStateVersionRequest builds the JSONAPI state version create request a Terraform state push
// sends, including json-state when jsonState is non-empty. Like the CLI, it computes md5 and serial from
// the state, which must be a base64-encoded state with serial 1.
func newCreateStateVersionRequest(t *testing.T, workspaceGID, state, jsonState string) *http.Request {
	decoded, err := base64.StdEncoding.DecodeString(state)
	require.NoError(t, err)

	options := &gotfe.StateVersionCreateOptions{
		Serial: ptr.Int64(1),
		MD5:    ptr.String(fmt.Sprintf("%x", md5.Sum(decoded))),
		State:  ptr.String(state),
	}
	if jsonState != "" {
		options.JSONState = ptr.String(jsonState)
	}
	return newStateVersionCreateRequest(t, workspaceGID, options)
}

// newStateVersionCreateRequest builds a state version create request from options.
func newStateVersionCreateRequest(t *testing.T, workspaceGID string, options *gotfe.StateVersionCreateOptions) *http.Request {
	var body bytes.Buffer
	require.NoError(t, jsonapi.MarshalPayload(&body, options))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("workspaceId", workspaceGID)
	req := httptest.NewRequest(http.MethodPost, "/workspaces/"+workspaceGID+"/state-versions", &body)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// TestWorkspaceController_CreateStateVersionAttributes covers checking md5, serial and lineage against the
// state they describe, which the Terraform CLI computes all three from, and the request size limit.
func TestWorkspaceController_CreateStateVersionAttributes(t *testing.T) {
	const (
		workspaceID    = "a1b2c3d4-1c39-4c8f-9a2f-2c4d5e6f7a80"
		stateVersionID = "b8e2f2b4-1c39-4c8f-9a2f-2c4d5e6f7a80"
	)

	rawState := []byte(`{"version":4,"serial":3,"lineage":"9b2c1e4a"}`)
	state := base64.StdEncoding.EncodeToString(rawState)
	stateMD5 := fmt.Sprintf("%x", md5.Sum(rawState))

	tests := []struct {
		name       string
		options    *gotfe.StateVersionCreateOptions
		maxSize    int
		expectCode int
	}{
		{
			name:       "md5, serial and lineage match the state",
			options:    &gotfe.StateVersionCreateOptions{MD5: ptr.String(stateMD5), Serial: ptr.Int64(3), Lineage: ptr.String("9b2c1e4a")},
			expectCode: http.StatusCreated,
		},
		{
			name:       "md5 is compared case-insensitively",
			options:    &gotfe.StateVersionCreateOptions{MD5: ptr.String(strings.ToUpper(stateMD5))},
			expectCode: http.StatusCreated,
		},
		{
			name:       "none sent",
			options:    &gotfe.StateVersionCreateOptions{},
			expectCode: http.StatusCreated,
		},
		{
			name:       "md5 of the base64 text is not the state's",
			options:    &gotfe.StateVersionCreateOptions{MD5: ptr.String(fmt.Sprintf("%x", md5.Sum([]byte(state))))},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "serial does not match",
			options:    &gotfe.StateVersionCreateOptions{MD5: ptr.String(stateMD5), Serial: ptr.Int64(4)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "negative serial",
			options:    &gotfe.StateVersionCreateOptions{Serial: ptr.Int64(-3)},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "lineage does not match",
			options:    &gotfe.StateVersionCreateOptions{Lineage: ptr.String("other")},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "state is not base64",
			options:    &gotfe.StateVersionCreateOptions{MD5: ptr.String(stateMD5), State: ptr.String("not base64!!")},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "state is not JSON",
			options:    &gotfe.StateVersionCreateOptions{Serial: ptr.Int64(3), State: ptr.String(base64.StdEncoding.EncodeToString([]byte("serial=3")))},
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "request over the size limit",
			options:    &gotfe.StateVersionCreateOptions{MD5: ptr.String(stateMD5)},
			maxSize:    64,
			expectCode: http.StatusRequestEntityTooLarge,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.options.State == nil {
				test.options.State = ptr.String(state)
			}
			if test.maxSize == 0 {
				test.maxSize = 1024 * 1024
			}

			mockWorkspaceService := workspaceservice.NewMockService(t)
			if test.expectCode == http.StatusCreated {
				mockWorkspaceService.On("CreateStateVersion", mock.Anything, mock.Anything, state, (*string)(nil)).
					Return(&models.StateVersion{Metadata: models.ResourceMetadata{ID: stateVersionID}, WorkspaceID: workspaceID}, nil)
			}

			testLogger, _ := logger.NewForTest()
			c := &workspaceController{
				respWriter:                response.NewWriter(testLogger),
				logger:                    testLogger,
				workspaceService:          mockWorkspaceService,
				stateVersionMaxUploadSize: test.maxSize,
			}

			rec := httptest.NewRecorder()
			c.CreateStateVersion(rec, newStateVersionCreateRequest(t, gid.ToGlobalID(types.WorkspaceModelType, workspaceID), test.options))
			assert.Equal(t, test.expectCode, rec.Code, rec.Body.String())
		})
	}
}

// TestWorkspaceController_GetWorkspaceRuns lists a workspace's runs through go-tfe's Runs.List, as the
// remote backend does to find a queued run's position, and checks each page and its pagination.
// fakeRunsPager emulates the run service's cursor pagination over runs. A cursor is the ID of the
// last run on the previous page.
func fakeRunsPager(runs []*models.Run, calls *[]*pagination.Options) func(context.Context, *runservice.GetRunsInput) (*db.RunsResult, error) {
	return func(_ context.Context, input *runservice.GetRunsInput) (*db.RunsResult, error) {
		if calls != nil {
			*calls = append(*calls, input.PaginationOptions)
		}
		start := 0
		if input.PaginationOptions.After != nil {
			for i, r := range runs {
				if r.Metadata.ID == *input.PaginationOptions.After {
					start = i + 1
				}
			}
		}
		end := min(start+int(*input.PaginationOptions.First), len(runs))
		return &db.RunsResult{
			Runs: runs[start:end],
			PageInfo: &pagination.PageInfo{
				TotalCount: pagination.StaticCount(int32(len(runs))),
				Cursor: func(cp pagination.CursorPaginatable) (*string, error) {
					id := cp.(*models.Run).Metadata.ID
					return &id, nil
				},
			},
		}, nil
	}
}

func TestWorkspaceController_GetWorkspaceRuns(t *testing.T) {
	const workspaceID = "a1b2c3d4-1c39-4c8f-9a2f-2c4d5e6f7a80"

	workspace := &models.Workspace{Metadata: models.ResourceMetadata{ID: workspaceID}, FullPath: "group/ws"}
	allRuns := make([]*models.Run, 5)
	for i := range allRuns {
		allRuns[i] = &models.Run{
			Metadata:    models.ResourceMetadata{ID: fmt.Sprintf("e5f6a7b8-1c39-4c8f-9a2f-2c4d5e6f7a8%d", i)},
			WorkspaceID: workspaceID,
			Status:      models.RunPlanned,
		}
	}

	tests := []struct {
		name             string
		options          *gotfe.RunListOptions
		expectFirsts     []int32
		expectRuns       []*models.Run
		expectPagination gotfe.Pagination
	}{
		{
			name:             "default page holds every run",
			options:          &gotfe.RunListOptions{},
			expectFirsts:     []int32{defaultRunsPageSize},
			expectRuns:       allRuns,
			expectPagination: gotfe.Pagination{CurrentPage: 1, TotalPages: 1, TotalCount: 5},
		},
		{
			name:             "first page",
			options:          &gotfe.RunListOptions{ListOptions: gotfe.ListOptions{PageSize: 2}},
			expectFirsts:     []int32{2},
			expectRuns:       allRuns[:2],
			expectPagination: gotfe.Pagination{CurrentPage: 1, NextPage: 2, TotalPages: 3, TotalCount: 5},
		},
		{
			name:             "middle page",
			options:          &gotfe.RunListOptions{ListOptions: gotfe.ListOptions{PageNumber: 2, PageSize: 2}},
			expectFirsts:     []int32{2, 2},
			expectRuns:       allRuns[2:4],
			expectPagination: gotfe.Pagination{CurrentPage: 2, PreviousPage: 1, NextPage: 3, TotalPages: 3, TotalCount: 5},
		},
		{
			name:             "last page is short",
			options:          &gotfe.RunListOptions{ListOptions: gotfe.ListOptions{PageNumber: 3, PageSize: 2}},
			expectFirsts:     []int32{4, 2},
			expectRuns:       allRuns[4:],
			expectPagination: gotfe.Pagination{CurrentPage: 3, PreviousPage: 2, TotalPages: 3, TotalCount: 5},
		},
		{
			name:             "page past the end is empty",
			options:          &gotfe.RunListOptions{ListOptions: gotfe.ListOptions{PageNumber: 4, PageSize: 2}},
			expectFirsts:     []int32{6},
			expectPagination: gotfe.Pagination{CurrentPage: 4, PreviousPage: 3, TotalPages: 3, TotalCount: 5},
		},
		{
			name:             "page size is capped",
			options:          &gotfe.RunListOptions{ListOptions: gotfe.ListOptions{PageSize: 1000}},
			expectFirsts:     []int32{maxRunsPageSize},
			expectRuns:       allRuns,
			expectPagination: gotfe.Pagination{CurrentPage: 1, TotalPages: 1, TotalCount: 5},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaceService := workspaceservice.NewMockService(t)
			mockRunService := runservice.NewMockService(t)

			mockWorkspaceService.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(workspace, nil)
			var calls []*pagination.Options
			mockRunService.On("GetRuns", mock.Anything, mock.MatchedBy(func(input *runservice.GetRunsInput) bool {
				return input.Workspace == workspace && *input.Sort == db.RunSortableFieldCreatedAtDesc
			})).Return(fakeRunsPager(allRuns, &calls))

			testLogger, _ := logger.NewForTest()
			c := &workspaceController{
				respWriter:       response.NewWriter(testLogger),
				logger:           testLogger,
				runService:       mockRunService,
				workspaceService: mockWorkspaceService,
			}
			client := newGoTFEClient(t, c)

			list, err := client.Runs.List(context.Background(), workspace.GetGlobalID(), test.options)
			require.NoError(t, err)

			var gotFirsts []int32
			for _, o := range calls {
				gotFirsts = append(gotFirsts, *o.First)
			}
			assert.Equal(t, test.expectFirsts, gotFirsts)

			require.Len(t, list.Items, len(test.expectRuns))
			for i, run := range test.expectRuns {
				assert.Equal(t, run.GetGlobalID(), list.Items[i].ID)
				assert.Equal(t, gotfe.RunPlanned, list.Items[i].Status)
			}

			require.NotNil(t, list.Pagination)
			assert.Equal(t, test.expectPagination, *list.Pagination)
		})
	}
}

// TestWorkspaceController_GetWorkspaceRunsInvalidPage checks a page parameter that is not a positive
// integer is the client's error.
func TestWorkspaceController_GetWorkspaceRunsInvalidPage(t *testing.T) {
	for _, query := range []string{"page[number]=0", "page[size]=-1", "page[number]=x", "page[size]=1.5", "page[number]=102&page[size]=100", "page[number]=2147483647"} {
		t.Run(query, func(t *testing.T) {
			testLogger, _ := logger.NewForTest()
			c := &workspaceController{respWriter: response.NewWriter(testLogger), logger: testLogger}

			router := chi.NewRouter()
			c.RegisterRoutes(router)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/workspaces/W_x/runs?"+query, nil))
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}
}
