package glob

import (
	"testing"
)

func TestShadows(t *testing.T) {
	type testCase struct {
		name string
		a    string
		b    string
		want bool
	}

	tests := []testCase{
		{
			name: "a is the wildcard -- always conflicts",
			a:    "*",
			b:    "anything-*-at-all",
			want: true,
		},
		{
			name: "b is the wildcard but a is not -- never conflicts",
			a:    "abc",
			b:    "*",
			want: false,
		},
		{
			name: "both are the wildcard -- conflicts",
			a:    "*",
			b:    "*",
			want: true,
		},
		{
			name: "both literal and equal -- conflicts",
			a:    "1.0.0",
			b:    "1.0.0",
			want: true,
		},
		{
			name: "both literal and different -- no conflict",
			a:    "1.0.0",
			b:    "1.0.1",
			want: false,
		},
		{
			name: "a literal, b wildcard -- never conflicts",
			a:    "1.0.0",
			b:    "1.0.*",
			want: false,
		},
		{
			name: "a wildcard, b literal that matches -- conflicts",
			a:    "1.0.*",
			b:    "1.0.0",
			want: true,
		},
		{
			name: "a wildcard, b literal that does not match -- no conflict",
			a:    "1.0.*",
			b:    "2.0.0",
			want: false,
		},
		{
			name: "the user's example -- *-* conflicts with *-rc.*",
			a:    "*-*",
			b:    "*-rc.*",
			want: true,
		},
		{
			name: "the reverse is not true -- *-rc.* does not conflict with *-*",
			a:    "*-rc.*",
			b:    "*-*",
			want: false,
		},
		{
			name: "exact duplicate patterns -- conflicts",
			a:    "*-*",
			b:    "*-*",
			want: true,
		},
		{
			name: "partial overlap, neither contains the other -- no conflict",
			a:    "1.*",
			b:    "*-rc.*",
			want: false,
		},
		{
			name: "a only reduces b's matches, does not cover all of them -- no conflict",
			a:    "*-alpha*",
			b:    "*-*",
			want: false,
		},
		{
			name: "start-anchored a requires start-anchored b with a matching prefix",
			a:    "ab*",
			b:    "abc*",
			want: true,
		},
		{
			name: "start-anchored a rejects an unanchored b",
			a:    "ab*",
			b:    "*abc*",
			want: false,
		},
		{
			name: "start-anchored a rejects a b with a different prefix",
			a:    "ab*",
			b:    "xy*",
			want: false,
		},
		{
			name: "end-anchored a requires end-anchored b with a matching suffix",
			a:    "*cd",
			b:    "*xcd",
			want: true,
		},
		{
			name: "end-anchored a rejects an unanchored b",
			a:    "*cd",
			b:    "*cd*",
			want: false,
		},
		{
			name: "both ends anchored -- conflict via a shared middle block",
			a:    "ab*ab",
			b:    "ab*xab",
			want: true,
		},
		{
			name: "both ends anchored -- the required suffix is missing",
			a:    "ab*ab",
			b:    "ab*xa",
			want: false,
		},
		{
			name: "unanchored single segment -- found anywhere",
			a:    "*mid*",
			b:    "*xmidy*",
			want: true,
		},
		{
			name: "unanchored single segment -- not found anywhere",
			a:    "*mid*",
			b:    "*xyz*",
			want: false,
		},
		{
			name: "three segments -- middle segment found inside one of b's blocks",
			a:    "a*mid*b",
			b:    "a*xmidy*b",
			want: true,
		},
		{
			name: "three segments -- middle segment would require spanning two of b's blocks",
			a:    "a*mid*b",
			b:    "a*mi*db",
			want: false,
		},
		{
			name: "empty pattern only matches the empty string",
			a:    "",
			b:    "",
			want: true,
		},
		{
			name: "empty a does not conflict with a non-empty literal b",
			a:    "",
			b:    "x",
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Shadows(test.a, test.b)
			if got != test.want {
				t.Errorf("Conflicts(%q, %q) = %v, want %v", test.a, test.b, got, test.want)
			}
		})
	}
}

func TestGlob(t *testing.T) {
	type testCase struct {
		name    string
		pattern string
		value   string
		want    bool
	}

	tests := []testCase{
		{
			name:    "wildcard matches anything",
			pattern: "*",
			value:   "anything",
			want:    true,
		},
		{
			name:    "empty pattern only matches an empty value, unlike CleanupGlob.Matches",
			pattern: "",
			value:   "anything",
			want:    false,
		},
		{
			name:    "empty pattern matches an empty value",
			pattern: "",
			value:   "",
			want:    true,
		},
		{
			name:    "literal pattern matches the identical value",
			pattern: "1.0.0",
			value:   "1.0.0",
			want:    true,
		},
		{
			name:    "wildcard pattern matches a value with the required substring",
			pattern: "*-rc.*",
			value:   "1.0.0-rc.1",
			want:    true,
		},
		{
			name:    "wildcard pattern does not match a value missing the required substring",
			pattern: "*-rc.*",
			value:   "1.0.0",
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Glob(test.pattern, test.value)
			if got != test.want {
				t.Errorf("Glob(%q, %q) = %v, want %v", test.pattern, test.value, got, test.want)
			}
		})
	}
}
