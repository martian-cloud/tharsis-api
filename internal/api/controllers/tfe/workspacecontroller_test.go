package tfe

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	workspaceservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
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
// sends, including json-state when jsonState is non-empty.
func newCreateStateVersionRequest(t *testing.T, workspaceGID, state, jsonState string) *http.Request {
	options := &gotfe.StateVersionCreateOptions{
		Serial: ptr.Int64(1),
		MD5:    ptr.String("d41d8cd98f00b204e9800998ecf8427e"),
		State:  ptr.String(state),
	}
	if jsonState != "" {
		options.JSONState = ptr.String(jsonState)
	}

	var body bytes.Buffer
	require.NoError(t, jsonapi.MarshalPayload(&body, options))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("workspaceId", workspaceGID)
	req := httptest.NewRequest(http.MethodPost, "/workspaces/"+workspaceGID+"/state-versions", &body)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
