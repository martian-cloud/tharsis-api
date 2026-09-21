package smtp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestSMTPSendMail(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	// Port 0 on localhost is not listening, so DialAndSend fails; this exercises the wrapper's
	// error handling (message assembly + wrapped dial error) without standing up an SMTP server.
	provider := NewProvider(testLogger, "127.0.0.1", 0, "from@x.com", "user", "pass", true)

	err := provider.SendMail(t.Context(), "to@x.com", "subject", "<p>body</p>", "corr-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}
