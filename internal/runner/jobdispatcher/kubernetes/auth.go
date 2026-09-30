package kubernetes

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer/cert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer/configfile"
	ekscfg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer/eks"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer/idtoken"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer/incluster"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
)

// Auth Types
const (
	AuthTypeEKSIAM        = "eks_iam"
	AuthTypeKubeConfig    = "kube_config"
	AuthTypeX509Cert      = "x509_cert"
	AuthTypeRunnerIDToken = "runner_id_token"
	AuthTypeInCluster     = "in_cluster"
)

var (
	requireEKSIAMAuthFields        = []string{"region", "eks_cluster"}
	requireKubeConfigAuthFields    = []string{"kube_config_path"}
	requireX509CertAuthFields      = []string{"kube_server", "client_cert", "client_key"}
	requireRunnerIDTokenAuthFields = []string{"kube_server"}
)

// parseConfigurer builds the configurer for the configured auth type.
func parseConfigurer(ctx context.Context, pluginData map[string]string, tokenGetter types.TokenGetterFunc) (configurer.Configurer, error) {
	authType := pluginData["auth_type"]
	switch authType {
	case AuthTypeEKSIAM:
		if err := checkRequiredFields(AuthTypeEKSIAM, pluginData, requireEKSIAMAuthFields); err != nil {
			return nil, err
		}

		c, err := ekscfg.New(ctx, pluginData["region"], pluginData["eks_cluster"])
		if err != nil {
			return nil, fmt.Errorf("failed to configure kube job dispatcher plugin with auth type %q : %v", AuthTypeEKSIAM, err)
		}

		return c, nil
	case AuthTypeKubeConfig:
		if err := checkRequiredFields(AuthTypeKubeConfig, pluginData, requireKubeConfigAuthFields); err != nil {
			return nil, err
		}

		c, err := configfile.New(pluginData["kube_config_path"])
		if err != nil {
			return nil, fmt.Errorf("failed to configure kube job dispatcher plugin with auth type %q : %v", AuthTypeKubeConfig, err)
		}

		return c, nil
	case AuthTypeX509Cert:
		if err := checkRequiredFields(AuthTypeX509Cert, pluginData, requireX509CertAuthFields); err != nil {
			return nil, err
		}

		c, err := cert.New(pluginData["kube_server"], pluginData["client_cert"], pluginData["client_key"], pluginData["ca_cert"])
		if err != nil {
			return nil, fmt.Errorf("failed to configure kube job dispatcher plugin with auth type %q : %v", AuthTypeX509Cert, err)
		}

		return c, nil
	case AuthTypeRunnerIDToken:
		if err := checkRequiredFields(AuthTypeRunnerIDToken, pluginData, requireRunnerIDTokenAuthFields); err != nil {
			return nil, err
		}

		c, err := idtoken.New(pluginData["kube_server"], pluginData["ca_cert"], tokenGetter)
		if err != nil {
			return nil, fmt.Errorf("failed to configure kube job dispatcher plugin with auth type %q : %v", AuthTypeRunnerIDToken, err)
		}

		return c, nil
	case AuthTypeInCluster:
		return incluster.New(), nil
	default:
		return nil, fmt.Errorf("kubernetes job dispatcher doesn't support auth_type '%s'", authType)
	}
}

func checkRequiredFields(authType string, pluginData map[string]string, requiredFields []string) error {
	for _, field := range requiredFields {
		if _, ok := pluginData[field]; !ok {
			return fmt.Errorf("kubernetes job dispatcher requires plugin data %q field when using the %q auth type", field, authType)
		}
	}

	return nil
}
