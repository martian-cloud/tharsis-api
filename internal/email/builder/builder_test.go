package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailTypeNewBuilder(t *testing.T) {
	tests := []struct {
		name      string
		emailType EmailType
		want      EmailBuilder
		wantErr   bool
	}{
		{
			name:      "failed run",
			emailType: FailedRunEmailType,
			want:      &FailedRunEmail{},
		},
		{
			name:      "service account secret expiration",
			emailType: ServiceAccountSecretExpirationEmailType,
			want:      &ServiceAccountSecretExpirationEmail{},
		},
		{
			name:      "signing key decommission",
			emailType: SigningKeyDecommissionEmailType,
			want:      &SigningKeyDecommissionEmail{},
		},
		{
			name:      "membership change",
			emailType: MembershipChangeEmailType,
			want:      &MembershipChangeEmail{},
		},
		{
			name:      "announcement",
			emailType: AnnouncementEmailType,
			want:      &AnnouncementEmail{},
		},
		{
			name:      "unknown type returns an error",
			emailType: EmailType("does_not_exist"),
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := test.emailType.NewBuilder()

			if test.wantErr {
				assert.Error(t, err)
				assert.Nil(t, b)
				return
			}

			require.NoError(t, err)
			assert.IsType(t, test.want, b)
			// The builder's Type() must round-trip back to the requested email type.
			assert.Equal(t, test.emailType, b.Type())
		})
	}
}

// TestEmailTypesConstructable iterates the authoritative EmailTypes list so a new type added to the
// list without a NewBuilder case fails here rather than silently going unbuildable.
func TestEmailTypesConstructable(t *testing.T) {
	types := EmailTypes()
	require.NotEmpty(t, types)

	for _, et := range types {
		t.Run(string(et), func(t *testing.T) {
			b, err := et.NewBuilder()
			require.NoError(t, err)
			require.NotNil(t, b)
			assert.Equal(t, et, b.Type())
		})
	}
}

func TestEmailTypeTemplateFilename(t *testing.T) {
	tests := []struct {
		name      string
		emailType EmailType
		want      string
	}{
		{
			name:      "failed run",
			emailType: FailedRunEmailType,
			want:      "failed_run.tmpl",
		},
		{
			name:      "announcement",
			emailType: AnnouncementEmailType,
			want:      "announcement.tmpl",
		},
		{
			name:      "arbitrary value is suffixed with .tmpl",
			emailType: EmailType("custom"),
			want:      "custom.tmpl",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, test.emailType.TemplateFilename())
		})
	}
}

func TestRenderMarkdownToHTML(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		want        string
		notContains []string
	}{
		{
			name:   "inline formatting",
			source: "**bold** *em* `code`",
			want:   "<p><strong>bold</strong> <em>em</em> <code>code</code></p>\n",
		},
		{
			name:   "link",
			source: "[a link](https://example.com)",
			want:   `<p><a href="https://example.com">a link</a></p>` + "\n",
		},
		{
			name:   "single newline becomes a hard break",
			source: "line one\nline two",
			want:   "<p>line one<br>\nline two</p>\n",
		},
		{
			name:   "heading and paragraph are separate blocks",
			source: "## Heading\n\nBody.",
			want:   "<h2>Heading</h2>\n<p>Body.</p>\n",
		},
		{
			name:   "unordered list",
			source: "- one\n- two",
			want:   "<ul>\n<li>one</li>\n<li>two</li>\n</ul>\n",
		},
		{
			name:   "ordered list",
			source: "1. first\n2. second",
			want:   "<ol>\n<li>first</li>\n<li>second</li>\n</ol>\n",
		},
		{
			name:   "fenced code block keeps blank lines verbatim",
			source: "```\ncode a\n\ncode b\n```",
			want:   "<pre><code>code a\n\ncode b\n</code></pre>\n",
		},
		{
			name:   "consecutive blank lines collapse to one paragraph break",
			source: "a\n\n\n\nb",
			want:   "<p>a</p>\n<p>b</p>\n",
		},
		{
			name:        "raw HTML tags are neutralized",
			source:      "before <script>alert(1)</script> after",
			notContains: []string{"<script>", "</script>"},
		},
		{
			name:   "javascript URL scheme is stripped",
			source: "[x](javascript:alert(1))",
			want:   `<p><a href="">x</a></p>` + "\n",
		},
		{
			name:   "empty source renders nothing",
			source: "",
			want:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := RenderMarkdownToHTML(test.source)
			require.NoError(t, err)

			if test.want != "" {
				assert.Equal(t, test.want, string(got))
			}

			for _, notWant := range test.notContains {
				assert.NotContains(t, string(got), notWant)
			}
		})
	}
}
