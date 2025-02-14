// Package plugin package
package plugin

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/go-limiter/memorystore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/go-redisstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/plunk"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/ses"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/smtp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/ratelimitstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws/awskms"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws/memory"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore/aws"
)

// Catalog contains the available plugins
type Catalog struct {
	ObjectStore           objectstore.ObjectStore
	JWSProvider           jws.Provider
	GraphqlRateLimitStore ratelimitstore.Store
	HTTPRateLimitStore    ratelimitstore.Store
	EmailProvider         email.Provider
}

// NewCatalog creates a new Catalog
func NewCatalog(ctx context.Context, logger logger.Logger, cfg *config.Config) (*Catalog, error) {
	objectStore, err := newObjectStorePlugin(ctx, logger, cfg)
	if err != nil {
		return nil, err
	}

	jwsProvider, err := newJWSProviderPlugin(ctx, logger, cfg)
	if err != nil {
		return nil, err
	}

	graphqlRateLimitStore, err := newRateLimitStore(ctx, logger,
		cfg.RateLimitStorePluginType, cfg.RateLimitStorePluginData, cfg.MaxGraphQLComplexity)
	if err != nil {
		return nil, err
	}

	// An authenticated request count as 1 unit; an unauthenticated request generally counts for more than 1 unit.
	httpRateLimitStore, err := newRateLimitStore(ctx, logger,
		cfg.RateLimitStorePluginType, cfg.RateLimitStorePluginData, cfg.HTTPRateLimit)
	if err != nil {
		return nil, err
	}

	emailProvider, err := newEmailProvider(ctx, logger, cfg.EmailClientPluginType, cfg.EmailClientPluginData)
	if err != nil {
		return nil, err
	}

	return &Catalog{
		ObjectStore:           objectStore,
		JWSProvider:           jwsProvider,
		GraphqlRateLimitStore: graphqlRateLimitStore,
		HTTPRateLimitStore:    httpRateLimitStore,
		EmailProvider:         emailProvider,
	}, nil
}

// newRateLimiterStore takes config and determines the cache type
func newRateLimitStore(_ context.Context, logger logger.Logger, pluginType string, pluginData map[string]string,
	tokenLimit int) (ratelimitstore.Store, error) {
	tokenLimit64 := uint64(tokenLimit)

	switch pluginType {
	case "redis":
		endpoint, ok := pluginData["redis_endpoint"]
		if !ok {
			return nil, errors.New("'redis_endpoint' is required when using the redis rate limit store")
		}

		pool := &redis.Pool{
			MaxIdle:   80,
			MaxActive: 1000,
			Dial: func() (redis.Conn, error) {
				conn, err := redis.DialURL(endpoint, redis.DialConnectTimeout(time.Second*30))
				if err != nil {
					logger.Errorf("Failed to connect to redis rate limit store at endpoint %s: %v", endpoint, err)
					os.Exit(1)
				}
				return conn, err
			},
		}

		redis, err := redisstore.NewWithPool(&redisstore.Config{
			Tokens:   uint64(tokenLimit64),
			Interval: time.Second,
		}, pool)
		if err != nil {
			return nil, err
		}

		return redis, nil
	case "memory":
		store := &memorystore.Config{
			Tokens:   uint64(tokenLimit64),
			Interval: time.Second,
		}
		mem, err := memorystore.New(store)
		if err != nil {
			return nil, err
		}

		return mem, nil
	default:
		return nil, errors.New(
			"The specified rate limit store type %s is not currently supported",
			pluginType,
		)

	}
}

func newObjectStorePlugin(ctx context.Context, logger logger.Logger, cfg *config.Config) (objectstore.ObjectStore, error) {
	var (
		store objectstore.ObjectStore
		err   error
	)

	switch cfg.ObjectStorePluginType {
	case "aws_s3":
		store, err = aws.New(ctx, logger, cfg.ObjectStorePluginData)
	default:
		err = errors.New(
			"The specified object store %s is not currently supported", cfg.ObjectStorePluginType,
		)
	}

	return store, err
}

func newJWSProviderPlugin(ctx context.Context, _ logger.Logger, cfg *config.Config) (jws.Provider, error) {
	var (
		plugin jws.Provider
		err    error
	)

	switch cfg.JWSProviderPluginType {
	case "memory":
		plugin, err = memory.New(cfg.JWSProviderPluginData)
	case "awskms":
		plugin, err = awskms.New(ctx, cfg.JWSProviderPluginData)
	default:
		err = errors.New(
			"The specified JWS Provider plugin %s is not currently supported", cfg.JWSProviderPluginType,
		)
	}

	return plugin, err
}

func newEmailProvider(ctx context.Context, logger logger.Logger, pluginType string, pluginData map[string]string) (email.Provider, error) {
	switch pluginType {
	case "smtp":
		smtpHost, ok := pluginData["smtp_host"]
		if !ok {
			return nil, errors.New("'smtp_host' is required when using the smtp email client plugin")
		}
		smtpPortRaw, ok := pluginData["smtp_port"]
		if !ok {
			return nil, errors.New("'smtp_port' is required when using the smtp email client plugin")
		}
		fromAddress, ok := pluginData["from_address"]
		if !ok {
			return nil, errors.New("'from_address' is required when using the smtp email client plugin")
		}
		smtpUsername, ok := pluginData["smtp_username"]
		if !ok {
			return nil, errors.New("'smtp_username' is required when using the smtp email client plugin")
		}
		smtpPassword, ok := pluginData["smtp_password"]
		if !ok {
			return nil, errors.New("'smtp_password' is required when using the smtp email client plugin")
		}

		disableTLS := false

		disableTLSRaw, ok := pluginData["disable_tls"]
		if ok {
			val, err := strconv.ParseBool(disableTLSRaw)
			if err != nil {
				return nil, fmt.Errorf("failed to parse 'disable_tls option for smpt plugin: %v", err)
			}
			disableTLS = val
		}

		smtpPort, err := strconv.Atoi(smtpPortRaw)
		if err != nil {
			return nil, fmt.Errorf("port must be a valid integer: %v", err)
		}

		return smtp.NewProvider(logger, smtpHost, smtpPort, fromAddress, smtpUsername, smtpPassword, disableTLS), nil
	case "ses":
		fromAddress, ok := pluginData["from_address"]
		if !ok {
			return nil, errors.New("'from_address' is required when using the ses email client plugin")
		}
		awsConfigSetName, ok := pluginData["aws_configuration_set_name"]
		if !ok {
			return nil, errors.New("'aws_configuration_set_name' is required when using the ses email client plugin")
		}
		region, ok := pluginData["region"]
		if !ok {
			return nil, errors.New("'region' is required when using the ses email client plugin")
		}

		sesClient, err := ses.NewProvider(ctx, logger, fromAddress, awsConfigSetName, region)
		if err != nil {
			return nil, fmt.Errorf("failed to load ses email plugin: %v", err)
		}

		return sesClient, nil
	case "plunk":
		endpoint, ok := pluginData["endpoint"]
		if !ok {
			return nil, errors.New("'endpoint' is required when using the plunk email client plugin")
		}
		apiKey, ok := pluginData["api_key"]
		if !ok {
			return nil, errors.New("'api_key' is required when using the plunk email client plugin")
		}
		return plunk.NewProvider(logger, endpoint, apiKey), nil
	case "":
		return &email.NoopProvider{}, nil
	default:
		return nil, fmt.Errorf("the specified email client plugin %s is not currently supported", pluginType)
	}
}
