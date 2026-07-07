package controllers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	jobservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	runservice "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestGetRunNodeLogs(t *testing.T) {
	const (
		runID  = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
		planID = "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"
		jobID  = "c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"
		token  = "the-token"
	)
	runGID := gid.ToGlobalID(types.RunModelType, runID)
	planGID := gid.ToGlobalID(types.PlanModelType, planID)

	logData := []byte("hello logs")

	tests := []struct {
		name              string
		nodePath          string
		latestJobID       *string
		applyNil          bool
		tokenValid        bool
		expectReadLogs    bool
		expectGetRun      bool
		expectVerifyToken bool
		wantStatus        int
		wantBody          string
	}{
		{
			name:              "plan node without a job yet returns empty 200 so the CLI keeps polling",
			nodePath:          models.PlanNodePath,
			latestJobID:       nil,
			tokenValid:        true,
			expectGetRun:      true,
			expectVerifyToken: true,
			wantStatus:        http.StatusOK,
			wantBody:          "",
		},
		{
			name:              "plan node with a job streams its logs",
			nodePath:          models.PlanNodePath,
			latestJobID:       ptrString(jobID),
			tokenValid:        true,
			expectGetRun:      true,
			expectVerifyToken: true,
			expectReadLogs:    true,
			wantStatus:        http.StatusOK,
			wantBody:          string(logData),
		},
		{
			name:              "invalid token is rejected before the run is queried",
			nodePath:          models.PlanNodePath,
			latestJobID:       ptrString(jobID),
			tokenValid:        false,
			expectVerifyToken: true,
			wantStatus:        http.StatusUnauthorized,
		},
		{
			name:              "speculative run with no apply node returns empty 200",
			nodePath:          models.ApplyNodePath,
			applyNil:          true,
			tokenValid:        true,
			expectGetRun:      true,
			expectVerifyToken: true,
			wantStatus:        http.StatusOK,
			wantBody:          "",
		},
		{
			name:       "unknown node path is rejected before resolving the run",
			nodePath:   "bogus",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockJobService := jobservice.NewMockService(t)
			mockRunService := runservice.NewMockService(t)
			mockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			if tt.expectGetRun {
				run := &models.Run{
					Metadata: models.ResourceMetadata{ID: runID},
					Plan:     models.Plan{ID: planID, LatestJobID: tt.latestJobID},
				}
				if !tt.applyNil {
					run.Apply = &models.Apply{ID: planID, LatestJobID: tt.latestJobID}
				}
				mockRunService.On("GetRunByID", mock.Anything, runID).Return(run, nil)
			}

			if tt.expectVerifyToken {
				// Assert the token is verified against the run's global ID (the URL run
				// param) as the subject — the core security property of this endpoint.
				// A regression that passed a different subject would fail to match here.
				subjectIsRunGID := tokenSubjectMatcher(runGID)
				if tt.tokenValid {
					mockSigningKeyManager.On("VerifyToken", mock.Anything, token, subjectIsRunGID).
						Return(&auth.VerifyTokenOutput{}, nil)
				} else {
					mockSigningKeyManager.On("VerifyToken", mock.Anything, token, subjectIsRunGID).
						Return(nil, errors.New("bad token", errors.WithErrorCode(errors.EUnauthorized)))
				}
			}

			if tt.expectReadLogs {
				mockJobService.On("ReadLogs", mock.Anything, jobID, 0, defaultLogReadLimit).
					Return(io.NopCloser(bytes.NewReader(logData)), nil)
			}

			testLogger, _ := logger.NewForTest()
			c := NewJobController(
				testLogger,
				response.NewWriter(testLogger),
				nil, // jwtAuthMiddleware unused: this route relies on the path token
				mockSigningKeyManager,
				mockJobService,
				mockRunService,
			)

			router := chi.NewRouter()
			c.RegisterRoutes(router)

			url := "/runs/" + runGID + "/" + tt.nodePath + "/logs/" + token
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, tt.wantBody, rec.Body.String())
			}
		})
	}

	// Sanity check that the global IDs we build round-trip back to the raw IDs the
	// handler resolves them to.
	assert.Equal(t, runID, gid.FromGlobalID(runGID))
	assert.Equal(t, planID, gid.FromGlobalID(planGID))
}

// tokenSubjectMatcher returns a testify argument matcher that accepts a
// jwt.ValidateOption only when it enforces the given subject. It works by applying
// the option to a token whose subject is wantSubject: jwt.Validate succeeds iff the
// option's subject matches, so this asserts the handler scoped the token to that
// subject (the run global ID) rather than some other value.
func tokenSubjectMatcher(wantSubject string) interface{} {
	return mock.MatchedBy(func(opt jwt.ValidateOption) bool {
		tok := jwt.New()
		if err := tok.Set(jwt.SubjectKey, wantSubject); err != nil {
			return false
		}
		return jwt.Validate(tok, opt) == nil
	})
}

// TestGetRunNodeLogs_TokenScopedToDifferentRun verifies that a token issued for one run
// is rejected when presented at another run's log URL: the handler verifies the token
// against the URL's run global ID, so the signing key manager (here standing in for the
// real subject check) rejects it and the run is never queried.
func TestGetRunNodeLogs_TokenScopedToDifferentRun(t *testing.T) {
	const token = "token-for-run-a"
	runBGID := gid.ToGlobalID(types.RunModelType, "d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14")

	mockJobService := jobservice.NewMockService(t)
	mockRunService := runservice.NewMockService(t)
	mockSigningKeyManager := auth.NewMockSigningKeyManager(t)

	// The token's real subject is run A, so verifying it against run B's GID fails.
	mockSigningKeyManager.On("VerifyToken", mock.Anything, token, tokenSubjectMatcher(runBGID)).
		Return(nil, errors.New("subject mismatch", errors.WithErrorCode(errors.EUnauthorized)))

	testLogger, _ := logger.NewForTest()
	c := NewJobController(
		testLogger,
		response.NewWriter(testLogger),
		nil,
		mockSigningKeyManager,
		mockJobService,
		mockRunService,
	)

	router := chi.NewRouter()
	c.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/runs/"+runBGID+"/"+models.PlanNodePath+"/logs/"+token, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	mockRunService.AssertNotCalled(t, "GetRunByID")
}

func ptrString(s string) *string { return &s }
