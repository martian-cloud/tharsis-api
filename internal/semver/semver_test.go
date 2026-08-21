package semver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExactVersion(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		want       string
		wantExact  bool
	}{
		{
			name:       "full semantic version",
			constraint: "1.0.0",
			want:       "1.0.0",
			wantExact:  true,
		},
		{
			name:       "partial version is normalized",
			constraint: "1.2",
			want:       "1.2.0",
			wantExact:  true,
		},
		{
			name:       "caret range is not exact",
			constraint: "^1.0",
			wantExact:  false,
		},
		{
			name:       "operator constraint is not exact",
			constraint: ">=1.0.0",
			wantExact:  false,
		},
		{
			name:       "wildcard is not exact",
			constraint: "1.x",
			wantExact:  false,
		},
		{
			name:       "empty string is not exact",
			constraint: "",
			wantExact:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, exact := ExactVersion(tt.constraint)
			assert.Equal(t, tt.wantExact, exact)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHighestMatching(t *testing.T) {
	tests := []struct {
		name       string
		versions   []string
		constraint string
		want       string
		wantFound  bool
	}{
		{
			name:       "empty constraint returns highest",
			versions:   []string{"1.0.0", "2.0.0", "1.5.0"},
			constraint: "",
			want:       "2.0.0",
			wantFound:  true,
		},
		{
			name:       "constraint selects highest match in range",
			versions:   []string{"1.0.0", "2.0.0", "1.5.0"},
			constraint: ">=1.0.0, <2.0.0",
			want:       "1.5.0",
			wantFound:  true,
		},
		{
			name:       "no version satisfies the constraint",
			versions:   []string{"1.0.0", "1.5.0"},
			constraint: ">3.0.0",
			wantFound:  false,
		},
		{
			name:       "unparseable versions are ignored",
			versions:   []string{"1.0.0", "not-a-version", "1.2.0"},
			constraint: "",
			want:       "1.2.0",
			wantFound:  true,
		},
		{
			name:       "release outranks prerelease",
			versions:   []string{"1.0.0-rc1", "1.0.0"},
			constraint: "",
			want:       "1.0.0",
			wantFound:  true,
		},
		{
			name:       "invalid constraint returns not found",
			versions:   []string{"1.0.0"},
			constraint: "not-a-constraint",
			wantFound:  false,
		},
		{
			name:       "empty version list returns not found",
			versions:   nil,
			constraint: "",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := HighestMatching(tt.versions, tt.constraint)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.want, got)
		})
	}
}
