package serviceaccount

import (
	"context"
	"strings"
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

func FuzzClaimValueValidator(f *testing.F) {
	f.Add("sub", "repo:org/project:ref:main", "repo:org/project:ref:main", false)
	f.Add("sub", "repo:org/project:ref:main", "repo:org/*", true)
	f.Add("sub", "repo:org/project:ref:main", "repo:org/*", false)
	f.Add("sub", "repo:org/*", "repo:org/*", false)
	f.Add("sub", "repo:evil/org/project", "repo:org/*", true)
	f.Add("aud", "tharsis", "tharsis", false)
	f.Add("aud", "tharsis", "*", true)
	f.Add("my_claim", "", "", false)
	f.Add("my_claim", "", "*", true)
	f.Add("my_claim", "abc", "a*c", true)
	f.Add("my_claim", "aba", "ab*ba", true)
	f.Add("exp", "123", "123", false)

	f.Fuzz(func(t *testing.T, name, claim, value string, useGlob bool) {
		ctx := context.Background()

		token := jwt.New()
		if err := token.Set(name, claim); err != nil {
			t.Skip()
		}

		// A claim that is not present in the token never matches, not even a "*" glob.
		const absentClaim = "tharsis_fuzz_absent_claim"
		if name != absentClaim {
			assert.NotNil(t, newClaimValueValidator(absentClaim, value, useGlob).Validate(ctx, token))
		}

		err := newClaimValueValidator(name, value, useGlob).Validate(ctx, token)
		matched := err == nil

		var actual string
		if name == "aud" {
			aud := token.Audience()
			require.Len(t, aud, 1)
			actual = aud[0]
		} else {
			v, ok := token.Get(name)
			require.True(t, ok)
			s, isString := v.(string)
			if !isString {
				// Some registered claims (exp, iat, nbf) are parsed into non-string types and
				// are rejected rather than compared.
				assert.False(t, matched)
				return
			}
			actual = s
		}

		if !useGlob || !strings.Contains(value, "*") {
			// Exact matching (and glob patterns without wildcards) only accepts equal values.
			assert.Equal(t, actual == value, matched, "claim %q, value %q", actual, value)
			return
		}

		// A pattern always matches its own literal text.
		if actual == value {
			assert.True(t, matched)
		}

		parts := strings.Split(value, "*")

		// "prefix*" (single trailing wildcard) matches exactly the claims with that prefix.
		if len(parts) == 2 && parts[1] == "" {
			assert.Equal(t, strings.HasPrefix(actual, parts[0]), matched, "claim %q, value %q", actual, value)
		}

		if matched {
			// Every accepted claim contains the pattern's literal segments in order, starting
			// with the first segment and ending with the last one.
			assert.True(t, strings.HasPrefix(actual, parts[0]), "claim %q, value %q", actual, value)
			assert.True(t, strings.HasSuffix(actual, parts[len(parts)-1]), "claim %q, value %q", actual, value)
			assert.GreaterOrEqual(t, len(actual), len(value)-strings.Count(value, "*"))
			rest := actual
			for _, part := range parts {
				idx := strings.Index(rest, part)
				require.GreaterOrEqual(t, idx, 0, "claim %q, value %q", actual, value)
				rest = rest[idx+len(part):]
			}
		}
	})
}
