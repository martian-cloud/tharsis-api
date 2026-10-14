package servers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestAuthSettingsServer_GetAuthSettings(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		expectType   pb.UserAuthType
		expectIssuer string
		expectClient string
	}{
		{
			name: "oauth provider configured points the cli at the api",
			cfg: &config.Config{
				TharsisAPIURL: "https://api.example.com",
				OauthProviders: []config.IdpConfig{
					{IssuerURL: "https://idp.example.com", ClientID: "idp-client"},
				},
			},
			expectType:   pb.UserAuthType_OIDC,
			expectIssuer: "https://api.example.com",
			expectClient: "tharsis",
		},
		{
			name: "no oauth provider",
			cfg: &config.Config{
				TharsisAPIURL: "https://api.example.com",
			},
			expectType: pb.UserAuthType_BASIC,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, err := NewAuthSettingsServer(test.cfg).GetAuthSettings(context.Background(), &emptypb.Empty{})
			require.NoError(t, err)

			assert.Equal(t, test.expectType, resp.AuthType)
			if test.expectIssuer == "" {
				assert.Nil(t, resp.OidcAuthSettings)
				return
			}
			require.NotNil(t, resp.OidcAuthSettings)
			assert.Equal(t, test.expectIssuer, resp.OidcAuthSettings.IssuerUrl)
			assert.Equal(t, test.expectClient, resp.OidcAuthSettings.ClientId)
		})
	}
}
