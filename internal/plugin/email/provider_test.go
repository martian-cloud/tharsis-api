package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoopProviderSendMail(t *testing.T) {
	var p Provider = &NoopProvider{}

	// The noop provider accepts any send and never errors.
	assert.NoError(t, p.SendMail(t.Context(), "to@x.com", "subject", "body", "corr-1"))
}
