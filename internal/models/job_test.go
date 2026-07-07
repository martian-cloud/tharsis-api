package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJob_SetStatus_InitialAssignment verifies that the first assignment from the
// zero value (constructing a job or hydrating it from the database) is always
// allowed, regardless of the target status.
func TestJob_SetStatus_InitialAssignment(t *testing.T) {
	for _, status := range []JobStatus{JobQueued, JobPending, JobRunning, JobFinished, JobFailed, JobCanceled, JobCanceling} {
		job := &Job{}

		require.NoError(t, job.SetStatus(status))
		assert.Equal(t, status, job.GetStatus())
	}
}

// TestJob_SetStatus_ValidTransitions verifies that each transition in the job
// lifecycle is accepted.
func TestJob_SetStatus_ValidTransitions(t *testing.T) {
	tests := []struct {
		from JobStatus
		to   JobStatus
	}{
		{JobQueued, JobPending},
		{JobQueued, JobCanceled},
		{JobPending, JobRunning},
		{JobPending, JobCanceled},
		{JobRunning, JobFinished},
		{JobRunning, JobFailed},
		{JobRunning, JobCanceled},
		{JobRunning, JobCanceling},
		{JobCanceling, JobCanceled},
		{JobCanceling, JobFinished},
		{JobCanceling, JobFailed},
	}

	for _, tt := range tests {
		job := &Job{}
		require.NoError(t, job.SetStatus(tt.from))

		require.NoError(t, job.SetStatus(tt.to))
		assert.Equal(t, tt.to, job.GetStatus())
	}
}

// TestJob_SetStatus_InvalidTransitions verifies that transitions outside the job
// lifecycle are rejected and leave the status unchanged.
func TestJob_SetStatus_InvalidTransitions(t *testing.T) {
	tests := []struct {
		from JobStatus
		to   JobStatus
	}{
		{JobQueued, JobRunning},   // must be claimed (pending) first
		{JobQueued, JobFinished},  // cannot finish before running
		{JobPending, JobFinished}, // cannot finish before running
		{JobRunning, JobQueued},   // no going backwards
		{JobRunning, JobPending},  // no going backwards
		{JobFinished, JobRunning}, // final states are terminal
		{JobFailed, JobRunning},   // final states are terminal
		{JobCanceled, JobRunning}, // final states are terminal
		{JobCanceling, JobRunning},
		{JobRunning, JobRunning}, // already in this status
	}

	for _, tt := range tests {
		job := &Job{}
		require.NoError(t, job.SetStatus(tt.from))

		err := job.SetStatus(tt.to)

		require.Error(t, err)
		assert.Equal(t, tt.from, job.GetStatus(), "status should be unchanged after a rejected transition")
	}
}
