package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestRunGateApproval_Validate(t *testing.T) {
	userID := "user-1"
	serviceAccountID := "sa-1"

	tests := []struct {
		name             string
		decision         RunGateDecision
		userID           *string
		serviceAccountID *string
		expectErrorCode  errors.CodeType
	}{
		{
			name:     "approve by a user is valid",
			decision: RunGateDecisionApprove,
			userID:   &userID,
		},
		{
			name:     "reject by a user is valid",
			decision: RunGateDecisionReject,
			userID:   &userID,
		},
		{
			name:             "approve by a service account is valid",
			decision:         RunGateDecisionApprove,
			serviceAccountID: &serviceAccountID,
		},
		{
			name:            "unknown decision is invalid",
			decision:        RunGateDecision("maybe"),
			userID:          &userID,
			expectErrorCode: errors.EInvalid,
		},
		{
			name:             "both a user and a service account is invalid",
			decision:         RunGateDecisionApprove,
			userID:           &userID,
			serviceAccountID: &serviceAccountID,
			expectErrorCode:  errors.EInvalid,
		},
		{
			name:            "neither a user nor a service account is invalid",
			decision:        RunGateDecisionApprove,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approval := &RunGateApproval{
				Decision:         tt.decision,
				UserID:           tt.userID,
				ServiceAccountID: tt.serviceAccountID,
			}
			err := approval.Validate()
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
