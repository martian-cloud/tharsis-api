// Package plugin package
package plugin

import (
	"context"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/go-limiter/memorystore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/go-redisstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
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

	return &Catalog{
		ObjectStore:           objectStore,
		JWSProvider:           jwsProvider,
		GraphqlRateLimitStore: graphqlRateLimitStore,
		HTTPRateLimitStore:    httpRateLimitStore,
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
