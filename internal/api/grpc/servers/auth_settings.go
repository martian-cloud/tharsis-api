package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AuthSettingsServer embeds the UnimplementedAuthSettingsServer.
type AuthSettingsServer struct {
	pb.UnimplementedAuthSettingsServer
	cfg *config.Config
}

// NewAuthSettingsServer returns an instance of AuthSettingsServer.
func NewAuthSettingsServer(cfg *config.Config) *AuthSettingsServer {
	return &AuthSettingsServer{
		cfg: cfg,
	}
}

// GetAuthSettings returns the IDP auth settings used by the CLI.
//
// When an external identity provider is configured, the OIDC settings point at the API rather than the
// provider: the API proxies the login to the provider and issues a Tharsis user session token, so the CLI
// never talks to the provider directly.
func (s *AuthSettingsServer) GetAuthSettings(_ context.Context, _ *emptypb.Empty) (*pb.GetAuthSettingsResponse, error) {
	if len(s.cfg.OauthProviders) > 0 {
		return &pb.GetAuthSettingsResponse{
			AuthType: pb.UserAuthType_OIDC,
			OidcAuthSettings: &pb.OIDCAuthSettings{
				IssuerUrl: s.cfg.TharsisAPIURL,
				ClientId:  auth.TharsisOAuthClientID,
			},
		}, nil
	}

	return &pb.GetAuthSettingsResponse{
		AuthType: pb.UserAuthType_BASIC,
	}, nil
}
