package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkspaceAssessment_IsStaleInProgress(t *testing.T) {
	now := time.Now()
	old := now.Add(-2 * AssessmentStaleTimeout)
	recent := now.Add(-AssessmentStaleTimeout / 2)
	completed := now.Add(-time.Minute)

	testCases := []struct {
		name        string
		lastUpdated *time.Time
		completedAt *time.Time
		expect      bool
	}{
		{
			name:        "in progress and not updated within the stale timeout",
			lastUpdated: &old,
			completedAt: nil,
			expect:      true,
		},
		{
			name:        "in progress but updated recently",
			lastUpdated: &recent,
			completedAt: nil,
			expect:      false,
		},
		{
			name:        "completed assessment is never stale-in-progress",
			lastUpdated: &old,
			completedAt: &completed,
			expect:      false,
		},
		{
			name:        "missing last-updated timestamp is not treated as stale",
			lastUpdated: nil,
			completedAt: nil,
			expect:      false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			wa := &WorkspaceAssessment{
				Metadata:             ResourceMetadata{LastUpdatedTimestamp: test.lastUpdated},
				CompletedAtTimestamp: test.completedAt,
			}
			assert.Equal(t, test.expect, wa.IsStaleInProgress())
		})
	}
}
