package db

import (
	"strings"
	"testing"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestMembershipFilterByRootNamespaces(t *testing.T) {
	t.Run("empty memberships match nothing (literal false, not absent)", func(t *testing.T) {
		sql, _, err := dialect.From("namespaces").Select("path").
			Where(membershipFilterByRootNamespaces(nil)).ToSQL()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.ToLower(sql), "where false") {
			t.Fatalf("empty roots must produce WHERE false, got: %s", sql)
		}
	})

	t.Run("each root yields indexable self + descendant predicates and no LIKE ANY", func(t *testing.T) {
		sql, _, err := dialect.From("namespaces").Select("path").
			Where(membershipFilterByRootNamespaces([]models.MembershipNamespace{
				{Path: "acme"},
				{Path: "corp/team1"},
			})).ToSQL()
		if err != nil {
			t.Fatal(err)
		}
		low := strings.ToLower(sql)

		for _, want := range []string{
			`"namespaces"."path" = 'acme'`,
			`"namespaces"."path" like 'acme/%'`,
			`"namespaces"."path" = 'corp/team1'`,
			`"namespaces"."path" like 'corp/team1/%'`,
		} {
			if !strings.Contains(low, want) {
				t.Errorf("expected SQL to contain %q, got: %s", want, sql)
			}
		}

		// The whole point of the change: no LIKE ANY(subquery), which can't use an index.
		if strings.Contains(low, "any(") || strings.Contains(low, "any (") {
			t.Errorf("SQL must not use LIKE ANY, got: %s", sql)
		}
	})

	t.Run("LIKE metacharacters in a root path are escaped", func(t *testing.T) {
		// '_' is a LIKE single-char wildcard; an unescaped "team_a/%" prefix would
		// also match sibling trees such as "teamXa/...". The descendant LIKE must
		// carry the escaped path, while the exact-match Eq stays literal.
		sql, _, err := dialect.From("namespaces").Select("path").
			Where(membershipFilterByRootNamespaces([]models.MembershipNamespace{
				{Path: "team_a"},
			})).ToSQL()
		if err != nil {
			t.Fatal(err)
		}
		low := strings.ToLower(sql)

		if !strings.Contains(low, `like 'team\_a/%'`) {
			t.Errorf("descendant LIKE must escape the '_' wildcard, got: %s", sql)
		}
		if strings.Contains(low, `like 'team_a/%'`) {
			t.Errorf("descendant LIKE must not use the unescaped '_' wildcard, got: %s", sql)
		}
		if !strings.Contains(low, `"namespaces"."path" = 'team_a'`) {
			t.Errorf("exact-match Eq must stay literal (unescaped), got: %s", sql)
		}
	})
}
