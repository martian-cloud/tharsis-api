package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageVersion_GetSHASumHex(t *testing.T) {
	tests := []struct {
		name   string
		shaSum []byte
		want   string
	}{
		{
			name:   "encodes bytes as hex",
			shaSum: []byte{0xde, 0xad, 0xbe, 0xef},
			want:   "deadbeef",
		},
		{
			name:   "empty sum is empty string",
			shaSum: []byte{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := &PackageVersion{SHASum: tt.shaSum}
			assert.Equal(t, tt.want, pv.GetSHASumHex())
		})
	}
}

func TestPackageVersion_Validate(t *testing.T) {
	valid := func(mutate func(*PackageVersion)) *PackageVersion {
		pv := &PackageVersion{
			PackageID:       "package-1",
			SemanticVersion: "1.0.0",
			Status:          PackageVersionStatusPending,
		}
		if mutate != nil {
			mutate(pv)
		}
		return pv
	}

	tests := []struct {
		name    string
		pv      *PackageVersion
		wantErr bool
	}{
		{name: "valid", pv: valid(nil)},
		{name: "missing package", pv: valid(func(pv *PackageVersion) { pv.PackageID = "" }), wantErr: true},
		{name: "missing semantic version", pv: valid(func(pv *PackageVersion) { pv.SemanticVersion = "" }), wantErr: true},
		{name: "invalid semantic version", pv: valid(func(pv *PackageVersion) { pv.SemanticVersion = "not-a-version" }), wantErr: true},
		{name: "uploaded status is valid", pv: valid(func(pv *PackageVersion) { pv.Status = PackageVersionStatusUploaded })},
		{name: "unsupported status", pv: valid(func(pv *PackageVersion) { pv.Status = PackageVersionStatus("bogus") }), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pv.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
