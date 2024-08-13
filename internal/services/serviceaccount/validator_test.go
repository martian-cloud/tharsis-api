package serviceaccount

import (
	"context"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimValueValidator(t *testing.T) {
	type testCase struct {
		name             string
		useGlob          bool
		key              string
		value            string
		actualClaimValue interface{}
		match            bool
	}

	tests := []testCase{
		{
			name:             "string match",
			useGlob:          false,
			match:            true,
			key:              "sub",
			value:            "123",
			actualClaimValue: "123",
		},
		{
			name:    "string match for aud",
			useGlob: false,
			match:   true,
			key:     "aud",
			value:   "phobos",
		},
		{
			name:             "array match",
			useGlob:          false,
			match:            true,
			key:              "my_claim",
			value:            "[\"val1\",\"val2\"]",
			actualClaimValue: []string{"val1", "val2"},
		},
		{
			name:             "bool match",
			useGlob:          false,
			match:            true,
			key:              "my_claim",
			value:            "true",
			actualClaimValue: true,
		},
		{
			name:             "int match",
			useGlob:          false,
			match:            true,
			key:              "my_claim",
			value:            "100",
			actualClaimValue: 100,
		},
		{
			name:             "single wildcard string match",
			useGlob:          true,
			match:            true,
			key:              "my_claim",
			value:            "this/*",
			actualClaimValue: "this/is/a/test/value",
		},
		{
			name:             "multiple wildcard string match",
			useGlob:          true,
			match:            true,
			key:              "my_claim",
			value:            "this/*/test/*",
			actualClaimValue: "this/is/a/test/value",
		},
		{
			name:             "string should not match",
			useGlob:          false,
			match:            false,
			key:              "sub",
			value:            "123",
			actualClaimValue: "456",
		},
		{
			name:             "array does not match",
			useGlob:          false,
			match:            false,
			key:              "my_claim",
			value:            "[\"val1\",\"val3\"]",
			actualClaimValue: []string{"val1", "val2"},
		},
		{
			name:             "bool does not match",
			useGlob:          false,
			match:            false,
			key:              "my_claim",
			value:            "false",
			actualClaimValue: true,
		},
		{
			name:             "int does not match",
			useGlob:          false,
			match:            false,
			key:              "my_claim",
			value:            "10",
			actualClaimValue: 100,
		},
		{
			name:    "string should not match for aud",
			useGlob: false,
			match:   false,
			key:     "aud",
			value:   "not-phobos",
		},
		{
			name:             "single wildcard string should not match",
			useGlob:          true,
			match:            false,
			key:              "my_claim",
			value:            "this/*/test",
			actualClaimValue: "this/is/a/test/value",
		},
		{
			name:             "multiple wildcard string should not match",
			useGlob:          true,
			match:            false,
			key:              "my_claim",
			value:            "invalid/this/*/test/*",
			actualClaimValue: "this/is/a/test/value",
		},
		{
			name:             "wildcard string should not match when glob is false",
			useGlob:          false,
			match:            false,
			key:              "my_claim",
			value:            "*",
			actualClaimValue: "abc*",
		},
		{
			name:             "wildcard string should match exact string when wildcard is false",
			useGlob:          false,
			match:            true,
			key:              "my_claim",
			value:            "abc*",
			actualClaimValue: "abc*",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			builder := jwt.NewBuilder().Audience([]string{"phobos"})
			if test.key != "aud" {
				builder = builder.Claim(test.key, test.actualClaimValue)
			}

			token, err := builder.Build()
			require.Nil(t, err)

			err = newClaimValueValidator(test.key, test.value, test.useGlob).Validate(ctx, token)

			if test.match {
				assert.Nil(t, err, "expected claim to match")
			} else {
				assert.NotNil(t, err, "expected claim to not match")
			}
		})
	}
}
