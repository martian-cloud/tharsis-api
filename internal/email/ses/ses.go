// Package ses defines the ses email plugin
package ses

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	timeout = 30 * time.Second
)

type sesProvider struct {
	fromAddress             string
	awsConfigurationSetName string
	logger                  logger.Logger
	awsClient               *ses.Client
}

// NewProvider returns a new provider instance
func NewProvider(
	ctx context.Context,
	logger logger.Logger,
	fromAddress string,
	awsConfigurationSetName string,
	region string,
) (email.Provider, error) {

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	awsClient := ses.NewFromConfig(awsCfg)

	return &sesProvider{
		fromAddress:             fromAddress,
		awsConfigurationSetName: awsConfigurationSetName,
		logger:                  logger,
		awsClient:               awsClient,
	}, nil
}

func (s *sesProvider) SendMail(ctx context.Context, to []string, subject, body string) error {
	_, err := s.awsClient.SendEmail(ctx, &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: to,
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(body),
				},
			},
			Subject: &types.Content{
				Data: aws.String(subject),
			},
		},
		Source:               aws.String(fmt.Sprintf("Tharsis <%s>", s.fromAddress)),
		ConfigurationSetName: aws.String(s.awsConfigurationSetName),
	})
	if err != nil {
		return err
	}

	return nil
}
