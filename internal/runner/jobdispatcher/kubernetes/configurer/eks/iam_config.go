// Package eks package
package eks

//go:generate mockery --name eksClient --inpackage --case underscore
//go:generate mockery --name presigner --inpackage --case underscore

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/smithy-go/transport/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
	"k8s.io/client-go/rest"
)

var _ configurer.Configurer = (*IAMConfig)(nil)

const (
	clusterNameHeader  = "x-k8s-aws-id"
	stsExpireParameter = "X-Amz-Expires"

	tokenPrefix             = "k8s-aws-v1." // #nosec G101 -- This is a false positive
	tokenExpirationDuration = 14 * time.Minute
)

type eksClient interface {
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
}

type presigner interface {
	PresignGetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// IAMConfig is an EKS configurer with IAM credentials
type IAMConfig struct {
	sync.Mutex
	tokenExpiresAt  time.Time
	presigner       presigner
	clusterName     string
	clusterEndpoint string
	token           string
	clusterCAData   []byte
}

// New generates a new EKS IAM Config struct that implements the configurer.Configurer.
func New(ctx context.Context, region, clusterName string) (*IAMConfig, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	cfg := &IAMConfig{
		presigner:   sts.NewPresignClient(sts.NewFromConfig(awsCfg)),
		clusterName: clusterName,
	}

	// We can retrieve this data because it is long lived and to verify we have access to the cluster
	endpoint, caData, err := getClusterInfo(ctx, eks.NewFromConfig(awsCfg), clusterName)
	if err != nil {
		return nil, err
	}

	cfg.clusterEndpoint = endpoint
	cfg.clusterCAData = caData

	return cfg, nil
}

// getClusterInfo will get the cluster host and CA Data for calling the kubernetes cluster
func getClusterInfo(ctx context.Context, eksClient eksClient, name string) (string, []byte, error) {
	cluster, err := eksClient.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &name})
	if err != nil {
		return "", nil, err
	}

	if cluster.Cluster.CertificateAuthority == nil || cluster.Cluster.CertificateAuthority.Data == nil {
		return "", nil, fmt.Errorf("certificate authority was missing from EKS Cluster")
	}

	caData, err := base64.StdEncoding.DecodeString(*cluster.Cluster.CertificateAuthority.Data)
	if err != nil {
		return "", nil, err
	}

	return *cluster.Cluster.Endpoint, caData, nil
}

// GetConfig implements the configurer.Configurer interface by generating a rest config for the kubernetes client
func (c *IAMConfig) GetConfig(ctx context.Context) (*rest.Config, error) {
	cfg := &rest.Config{
		Host: c.clusterEndpoint,
	}
	cfg.CAData = c.clusterCAData

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	if c.token == "" || time.Now().After(c.tokenExpiresAt) {
		var err error
		cfg.BearerToken, err = c.getToken(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		cfg.BearerToken = c.token
	}

	return cfg, nil
}

// getToken will generate a bearer token for calling an EKS cluster by pre-signing a get caller identity request and encoding it
func (c *IAMConfig) getToken(ctx context.Context) (string, error) {
	req, err := c.presigner.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}, func(po *sts.PresignOptions) {
		po.ClientOptions = append(po.ClientOptions, func(o *sts.Options) {
			o.APIOptions = append(o.APIOptions, func(s *middleware.Stack) error {
				return s.Build.Add(middleware.BuildMiddlewareFunc(
					"EKSClusterHeader",
					func(ctx context.Context, in middleware.BuildInput, next middleware.BuildHandler) (middleware.BuildOutput, middleware.Metadata, error) {
						switch v := in.Request.(type) {
						case *http.Request:
							v.Header.Add(clusterNameHeader, c.clusterName)

							query := v.URL.Query()
							query.Set(stsExpireParameter, "60")
							v.URL.RawQuery = query.Encode()

							v.Method = "GET"
						}
						return next.HandleBuild(ctx, in)
					},
				), middleware.Before)
			})
		})
	})
	if err != nil {
		return "", err
	}

	// Set the expiration for the token
	c.tokenExpiresAt = time.Now().Add(tokenExpirationDuration)

	return tokenPrefix + strings.TrimRight(base64.URLEncoding.EncodeToString([]byte(req.URL)), "="), nil
}
