package sqltag

import "testing"

func TestInjectParseRoundTrip(t *testing.T) {
	const op = "jobs.getJob"
	sql := "SELECT * FROM jobs WHERE id = $1"

	tagged := Inject(op, sql)
	if got := Parse(tagged); got != op {
		t.Fatalf("Parse(Inject(...)) = %q, want %q", got, op)
	}
	if got := tagged[len(tagged)-len(sql):]; got != sql {
		t.Fatalf("injected SQL %q does not end with original SQL %q", tagged, sql)
	}
}

func TestParseUntagged(t *testing.T) {
	cases := []string{
		"SELECT 1",
		"",
		"/* something else */ SELECT 1",
		"/* op='unterminated SELECT 1",
	}
	for _, c := range cases {
		if got := Parse(c); got != "" {
			t.Errorf("Parse(%q) = %q, want empty", c, got)
		}
	}
}

func TestCommentSanitizesOp(t *testing.T) {
	// An op containing the comment terminator or quote must not break the comment.
	tagged := Inject("evil'*/ DROP", "SELECT 1")
	if got := Parse(tagged); got != "evil DROP" {
		t.Fatalf("Parse = %q, want sanitized %q", got, "evil DROP")
	}
}
