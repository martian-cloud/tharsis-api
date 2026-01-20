package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AuthSettingsServer embeds the UnimplementedAuthSettingsServer.
type AuthSettingsServer struct {
	pb.UnimplementedAuthSettingsServer
	oauthProviders []config.IdpConfig
}

// NewAuthSettingsServer returns an instance of AuthSettingsServer.
func NewAuthSettingsServer(oauthProviders []config.IdpConfig) *AuthSettingsServer {
	return &AuthSettingsServer{
		oauthProviders: oauthProviders,
	}
}

// GetAuthSettings returns the IDP auth settings.
func (s *AuthSettingsServer) GetAuthSettings(_ context.Context, _ *emptypb.Empty) (*pb.GetAuthSettingsResponse, error) {
	if len(s.oauthProviders) > 0 {
		provider := s.oauthProviders[0]
		return &pb.GetAuthSettingsResponse{
			AuthType: pb.UserAuthType_OIDC,
			OidcAuthSettings: &pb.OIDCAuthSettings{
				IssuerUrl: provider.IssuerURL,
				ClientId:  provider.ClientID,
			},
		}, nil
	}

	return &pb.GetAuthSettingsResponse{
		AuthType: pb.UserAuthType_BASIC,
	}, nil
}
