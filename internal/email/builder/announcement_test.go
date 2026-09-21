package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnouncementEmailBuild(t *testing.T) {
	templateCtx := NewTemplateContext("https://tharsis.example.com", "")

	tests := []struct {
		name        string
		message     string
		severity    string
		contains    []string
		notContains []string
	}{
		{
			name:     "bold, emphasis, and inline code render as HTML",
			message:  "**bold** *em* and `code`",
			contains: []string{"<strong>bold</strong>", "<em>em</em>", "<code>code</code>"},
		},
		{
			name:     "links render with an href",
			message:  "[a link](https://example.com)",
			contains: []string{`<a href="https://example.com">a link</a>`},
		},
		{
			name:        "raw HTML in the source is not emitted as live markup (Unsafe off)",
			message:     "before <script>alert(1)</script> after",
			contains:    []string{"<!-- raw HTML omitted -->"},
			notContains: []string{"<script>", "</script>"},
		},
		{
			name:        "dangerous URL schemes are stripped to an empty href",
			message:     "[x](javascript:alert(1))",
			contains:    []string{`<a href="">x</a>`},
			notContains: []string{"javascript:"},
		},
		{
			name:     "a single newline becomes a hard line break",
			message:  "line one\nline two",
			contains: []string{"line one<br/>\nline two"},
		},
		{
			name:     "a heading and body render as separate blocks with scoped spacing",
			message:  "## Heading\n\nBody.",
			contains: []string{`<h2 style="margin:0 0 16px 0">Heading</h2>`, "Body."},
		},
		{
			name:     "an unordered list stays a single list with scoped spacing",
			message:  "intro\n\n- one\n- two\n\nend",
			contains: []string{`<ul style="margin:0 0 16px 0">`, "<li>one</li>", "<li>two</li>"},
		},
		{
			name:     "an ordered list renders as an ol",
			message:  "1. first\n2. second",
			contains: []string{"<ol", "<li>first</li>", "<li>second</li>"},
		},
		{
			name:        "a fenced code block keeps blank lines verbatim without injected markup",
			message:     "```\ncode a\n\ncode b\n```",
			contains:    []string{"<pre><code>code a\n\ncode b\n</code></pre>"},
			notContains: []string{"nbsp"},
		},
		{
			name:     "the last block drops its bottom margin so there is no trailing gap",
			message:  "First.\n\nLast.",
			contains: []string{`<p style="margin:0 0 16px 0;margin-bottom:0">Last.</p>`},
		},
		{
			name:     "consecutive blank lines collapse to uniform block spacing",
			message:  "a\n\n\n\nb",
			contains: []string{`<p style="margin:0 0 16px 0">a</p>`, `<p style="margin:0 0 16px 0;margin-bottom:0">b</p>`},
		},
		{
			name:     "an empty message still produces a valid email",
			message:  "",
			contains: []string{`class="text markdown-body"`},
		},
		{
			name:     "error severity uses the error alert container",
			message:  "boom",
			severity: "ERROR",
			contains: []string{`class="alert-error"`},
		},
		{
			name:     "warning severity uses the warning alert container",
			message:  "careful",
			severity: "WARNING",
			contains: []string{`class="alert-warning"`},
		},
		{
			name:     "success severity uses the success alert container",
			message:  "done",
			severity: "SUCCESS",
			contains: []string{`class="alert-success"`},
		},
		{
			name:     "an unknown severity falls back to the info alert container",
			message:  "hello",
			severity: "SOMETHING_ELSE",
			contains: []string{`class="alert-info"`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			severity := test.severity
			if severity == "" {
				severity = "INFO"
			}

			html, err := (&AnnouncementEmail{Message: test.message, Severity: severity}).Build(templateCtx)
			require.NoError(t, err)

			for _, want := range test.contains {
				assert.Contains(t, html, want)
			}

			for _, notWant := range test.notContains {
				assert.NotContains(t, html, notWant)
			}

			// Every announcement embeds the recipient-tokenized CTA back into the app.
			assert.Contains(t, html, "Open Dashboard")
			assert.Contains(t, html, `href="https://tharsis.example.com?rid=`+RecipientTokenPlaceholder+`"`)
		})
	}
}

func TestAnnouncementEmailBuildNilContext(t *testing.T) {
	_, err := (&AnnouncementEmail{Message: "x"}).Build(nil)
	assert.Error(t, err)
}
