package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestSCIMController_RespondWithSCIMError(t *testing.T) {
	tests := []struct {
		err          error
		name         string
		expectStatus int
		// expectScimType is the SCIM error type (RFC 7644 section 3.12) the response must carry.
		expectScimType string
	}{
		{
			name:           "a uniqueness conflict is reported as a uniqueness error",
			err:            errors.New("user already exists", errors.WithErrorCode(errors.EConflict)),
			expectStatus:   http.StatusConflict,
			expectScimType: "uniqueness",
		},
		{
			// A concurrent modification means the resource changed since it was read, not that a value
			// is already taken. Labeling it "uniqueness" would tell the identity provider the wrong thing.
			name:           "an optimistic lock error is a conflict that is not a uniqueness error",
			err:            errors.New("resource version does not match specified version", errors.WithErrorCode(errors.EOptimisticLock)),
			expectStatus:   http.StatusConflict,
			expectScimType: "",
		},
		{
			name:           "other errors carry no SCIM error type",
			err:            errors.New("not found", errors.WithErrorCode(errors.ENotFound)),
			expectStatus:   http.StatusNotFound,
			expectScimType: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testLogger, _ := logger.NewForTest()
			controller := &scimController{respWriter: response.NewWriter(testLogger), logger: testLogger}

			recorder := httptest.NewRecorder()
			controller.respondWithSCIMError(context.Background(), recorder, test.err)

			assert.Equal(t, test.expectStatus, recorder.Code)

			var body ScimErrorResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			assert.Equal(t, test.expectScimType, body.ScimType)
			// The SCIM status is the HTTP status as a string (the RFC requires a string).
			assert.Equal(t, strconv.Itoa(test.expectStatus), body.Status)
		})
	}
}
