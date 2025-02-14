// Package smtp defines the smtp email plugin
package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	gomail "gopkg.in/mail.v2"
)

const (
	timeout = 30 * time.Second
)

type smtpProvider struct {
	logger      logger.Logger
	dialer      *gomail.Dialer
	fromAddress string
}

// NewProvider returns a new provider instance
func NewProvider(
	logger logger.Logger,
	smtpHost string,
	smtpPort int,
	fromAddress string,
	username string,
	password string,
	disableTLS bool,
) email.Provider {
	dialer := gomail.NewDialer(smtpHost, smtpPort, username, password)

	dialer.Timeout = timeout
	if !disableTLS {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: false, MinVersion: tls.VersionTLS12, ServerName: smtpHost}
		dialer.StartTLSPolicy = gomail.MandatoryStartTLS
	} else {
		dialer.StartTLSPolicy = gomail.NoStartTLS
	}

	return &smtpProvider{
		fromAddress: fromAddress,
		logger:      logger,
		dialer:      dialer,
	}
}

func (s *smtpProvider) SendMail(_ context.Context, to []string, subject, body string) error {
	m := gomail.NewMessage()

	m.SetHeader("From", s.fromAddress)
	m.SetHeader("To", strings.Join(to, ", "))
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
