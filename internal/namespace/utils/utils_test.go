package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootGroupPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "nested path returns first segment",
			path: "root/sub/workspace",
			want: "root",
		},
		{
			name: "single segment returns itself",
			path: "root",
			want: "root",
		},
		{
			name: "empty path returns empty",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RootGroupPath(tt.path))
		})
	}
}
