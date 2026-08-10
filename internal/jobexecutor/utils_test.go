package jobexecutor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsOutputLimitViolation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "output limit violation error",
			err: status.Error(codes.InvalidArgument,
				"state version output limit exceeded: output \"big\" size 3145730 exceeds maximum allowed size of 2097152 bytes"),
			expected: true,
		},
		{
			name:     "other InvalidArgument error",
			err:      status.Error(codes.InvalidArgument, "invalid state data"),
			expected: false,
		},
		{
			name:     "unrelated Internal error",
			err:      status.Error(codes.Internal, "boom"),
			expected: false,
		},
		{
			name:     "plain non-gRPC error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, isOutputLimitViolation(test.err))
		})
	}
}
