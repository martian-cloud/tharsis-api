package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceLimits_AsEnvVars(t *testing.T) {
	tests := []struct {
		limits *ResourceLimits
		want   map[string]string
		name   string
	}{
		{
			name:   "nil receiver returns empty map",
			limits: nil,
			want:   map[string]string{},
		},
		{
			name:   "zero limits skip all vars",
			limits: &ResourceLimits{},
			want:   map[string]string{},
		},
		{
			name: "set limits render",
			limits: &ResourceLimits{
				MemoryBytes:          256,
				DiskReadBytes:        512,
				DiskWriteBytes:       1024,
				NetworkReceivedBytes: 2048,
				NetworkSentBytes:     4096,
			},
			want: map[string]string{
				"MEMORY_LIMIT":           "256",
				"DISK_READ_LIMIT":        "512",
				"DISK_WRITE_LIMIT":       "1024",
				"NETWORK_RECEIVED_LIMIT": "2048",
				"NETWORK_SENT_LIMIT":     "4096",
				"GOMEMLIMIT":             "230B", // 0.9 * 256
			},
		},
		{
			name:   "zero-valued limit is omitted",
			limits: &ResourceLimits{MemoryBytes: 256, DiskReadBytes: 0},
			want:   map[string]string{"MEMORY_LIMIT": "256", "GOMEMLIMIT": "230B"},
		},
		{
			name:   "GOMEMLIMIT is omitted when no memory limit is set",
			limits: &ResourceLimits{DiskReadBytes: 512},
			want:   map[string]string{"DISK_READ_LIMIT": "512"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.limits.AsEnvVars())
		})
	}
}

func TestLoadResourceLimits(t *testing.T) {
	tests := []struct {
		pluginData map[string]string
		want       *ResourceLimits
		name       string
		wantErr    bool
	}{
		{
			name:       "empty plugin data",
			pluginData: map[string]string{},
			want:       &ResourceLimits{},
		},
		{
			name:       "empty string skips the field",
			pluginData: map[string]string{"memory_limit": ""},
			want:       &ResourceLimits{},
		},
		{
			name:       "humanized sizes parse",
			pluginData: map[string]string{"memory_limit": "512Mi", "disk_read_limit": "1Gi"},
			want:       &ResourceLimits{MemoryBytes: 512 * 1024 * 1024, DiskReadBytes: 1024 * 1024 * 1024},
		},
		{
			name:       "unparseable value errors",
			pluginData: map[string]string{"memory_limit": "not-a-size"},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadResourceLimits(tt.pluginData)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
