package apikey

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGenerate(t *testing.T) {
	for _, keyType := range []Type{TypeSCIM, TypeVCS} {
		t.Run(string(keyType), func(t *testing.T) {
			key, hash, err := Generate(keyType)
			require.NoError(t, err)

			assert.Len(t, key, 53)
			assert.True(t, strings.HasPrefix(key, "ts"+string(keyType)+"_"))
			assert.Equal(t, GetHash(key), hash)

			parsedType, parsedHash, err := Parse(key)
			require.NoError(t, err)
			assert.Equal(t, keyType, parsedType)
			assert.Equal(t, hash, parsedHash)
		})
	}

	t.Run("unsupported type", func(t *testing.T) {
		_, _, err := Generate(Type("z"))
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("keys are unique", func(t *testing.T) {
		first, _, err := Generate(TypeSCIM)
		require.NoError(t, err)
		second, _, err := Generate(TypeSCIM)
		require.NoError(t, err)
		assert.NotEqual(t, first, second)
	})
}

func TestChecksum(t *testing.T) {
	assert.Len(t, checksumOf(""), checksumPartLength)
	assert.Equal(t, "000000", checksumOf(""))
	assert.Len(t, checksumOf(strings.Repeat("a", randomPartLength)), checksumPartLength)
}

func TestParse(t *testing.T) {
	validKey, _, err := Generate(TypeSCIM)
	require.NoError(t, err)

	random := validKey[headerLength : headerLength+randomPartLength]
	badChecksumKey := validKey[:totalKeyLength-1] + "0"
	if validKey[totalKeyLength-1] == '0' {
		badChecksumKey = validKey[:totalKeyLength-1] + "1"
	}

	tests := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "too short", key: validKey[:totalKeyLength-1]},
		{name: "too long", key: validKey + "a"},
		{name: "wrong product prefix", key: "xx" + validKey[2:]},
		{name: "unknown type", key: "tsz_" + random + checksumOf(random)},
		{name: "missing underscore", key: "tsc-" + random + checksumOf(random)},
		{name: "bad checksum", key: badChecksumKey},
		{name: "invalid charset", key: "tsc_" + random[:42] + "-" + checksumOf(random[:42]+"-")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := Parse(test.key)
			assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
		})
	}
}

func TestIsAPIKey(t *testing.T) {
	assert.True(t, IsAPIKey("tsc_abc"))
	assert.True(t, IsAPIKey("tsv_"))
	assert.False(t, IsAPIKey("tsc"))
	assert.False(t, IsAPIKey("eyJhbGciOi"))
	assert.False(t, IsAPIKey(""))
}
