// Package plugin package
package plugin

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/go-limiter/memorystore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/go-redisstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jwsprovider"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jwsprovider/awskms"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jwsprovider/memory"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/objectstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/objectstore/aws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/ratelimitstore"
)

// Catalog contains the available plugins
type Catalog struct {
	ObjectStore    objectstore.ObjectStore
	JWSProvider    jwsprovider.JWSProvider
	RateLimitStore ratelimitstore.Store
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

	rateLimitStore, err := newRateLimitStore(ctx, logger, cfg)
	if err != nil {
		return nil, err
	}

	return &Catalog{
		ObjectStore:    objectStore,
		JWSProvider:    jwsProvider,
		RateLimitStore: rateLimitStore,
	}, nil
}

// newRateLimiterStore takes config and determines the cache type
func newRateLimitStore(_ context.Context, logger logger.Logger, cfg *config.Config) (ratelimitstore.Store, error) {
	switch cfg.RateLimitStorePluginType {
	case "redis":
		endpoint, ok := cfg.RateLimitStorePluginData["redis_endpoint"]
		if !ok {
			return nil, errors.NewError(errors.EInternal, "'redis_endpoint' is required when using the redis rate limit store")
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
			Tokens:   uint64(cfg.MaxGraphQLComplexity),
			Interval: time.Second,
		}, pool)
		if err != nil {
			return nil, err
		}

		return redis, nil
	case "memory":
		store := &memorystore.Config{
			Tokens:   uint64(cfg.MaxGraphQLComplexity),
			Interval: time.Second,
		}
		mem, err := memorystore.New(store)
		if err != nil {
			return nil, err
		}

		return mem, nil
	default:
		return nil, errors.NewError(
			errors.EInternal,
			"The specified rate limit store type %s is not currently supported",
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
		err = errors.NewError(
			errors.EInternal,
			fmt.Sprintf("The specified object store %s is not currently supported", cfg.ObjectStorePluginType),
		)
	}

	return store, err
}

func newJWSProviderPlugin(ctx context.Context, _ logger.Logger, cfg *config.Config) (jwsprovider.JWSProvider, error) {
	var (
		plugin jwsprovider.JWSProvider
		err    error
	)

	switch cfg.JWSProviderPluginType {
	case "memory":
		plugin, err = memory.New(cfg.JWSProviderPluginData)
	case "awskms":
		plugin, err = awskms.New(ctx, cfg.JWSProviderPluginData)
	default:
		err = errors.NewError(
			errors.EInternal,
			fmt.Sprintf("The specified JWS Provider plugin %s is not currently supported", cfg.JWSProviderPluginType),
		)
	}

	return plugin, err
}
