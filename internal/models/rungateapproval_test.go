package models

import (
	"testing"

	"github.com/aws/smithy-go/ptr"

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

// TestRunGateApproval_principalKey covers the fallback ordering. UserID and
// ServiceAccountID are the normal cases; the CreatedBy fallbacks cover a row whose
// principal was deleted or is malformed, and each must produce a distinct stable key so de-duplication in
// RunGate.Satisfied still treats one approver as one approver.
func TestRunGateApproval_principalKey(t *testing.T) {
	const id = "8f9a0b1c-2d3e-4f5a-6b7c-8d9e0f1a2b3c"

	tests := []struct {
		name     string
		approval RunGateApproval
		expected string
	}{
		{
			name:     "user id wins",
			approval: RunGateApproval{UserID: ptr.String(id), CreatedBy: SystemActor()},
			expected: "user:" + id,
		},
		{
			name:     "service account id wins over the actor",
			approval: RunGateApproval{ServiceAccountID: ptr.String(id), CreatedBy: SystemActor()},
			expected: "sa:" + id,
		},
		{
			name:     "a typed actor is preferred over its subject",
			approval: RunGateApproval{CreatedBy: NewUserActor(id).WithSubject("user@example.com")},
			expected: "actor:user:" + id,
		},
		{
			name:     "the subject of an untyped actor is the last resort",
			approval: RunGateApproval{CreatedBy: Actor{Subject: "user@example.com"}},
			expected: "createdBy:user@example.com",
		},
	}

	seen := map[string]string{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := test.approval.principalKey()
			assert.Equal(t, test.expected, key)

			// Keys must not collide across principal kinds.
			if prev, ok := seen[key]; ok {
				t.Fatalf("key %q already produced by %q", key, prev)
			}
			seen[key] = test.name
		})
	}
}
