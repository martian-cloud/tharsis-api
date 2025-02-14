// Package plunk defines the plunk email plugin
package plunk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	httpClientTimeout = 30 * time.Second
)

type sendEmailPayload struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

type plunkProvider struct {
	endpoint   string
	apiKey     string
	logger     logger.Logger
	httpClient *http.Client
}

// NewProvider returns a new provider instance
func NewProvider(logger logger.Logger, endpoint string, apiKey string) email.Provider {
	return &plunkProvider{
		endpoint:   endpoint,
		apiKey:     apiKey,
		logger:     logger,
		httpClient: &http.Client{Timeout: httpClientTimeout},
	}
}

func (p *plunkProvider) SendMail(ctx context.Context, to []string, subject, body string) error {
	payload, err := json.Marshal(&sendEmailPayload{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/v1/send", p.endpoint),
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to send email: status=%s response=%s", resp.Status, buf)
	}

	return nil
}
