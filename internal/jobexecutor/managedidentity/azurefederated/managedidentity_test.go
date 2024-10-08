package azurefederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientID := "client1"
	tenantID := "tenant1"

	authenticator, _ := New()

	defer authenticator.Close(ctx)

	dataBuffer, err := json.Marshal(&azurefederated.Data{ClientID: clientID, TenantID: tenantID})
	if err != nil {
		t.Fatal(err)
	}

	identity := types.ManagedIdentity{
		Metadata: types.ResourceMetadata{
			ID: "managedIdentity-1",
		},
		Data: base64.StdEncoding.EncodeToString(dataBuffer),
	}

	token := []byte("tokendata")

	env, err := authenticator.Authenticate(
		ctx,
		[]types.ManagedIdentity{identity},
		func(_ context.Context, _ *types.ManagedIdentity) ([]byte, error) {
			return token, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Because the temporary file path/name is generated inside the authenticator,
	// this has to query the returned environment variable to get to the file.
	filePath := env["AZURE_FEDERATED_TOKEN_FILE"]

	data, _ := os.ReadFile(filePath)
	assert.Equal(t, token, data)

	assert.Equal(t, map[string]string{
		"ARM_TENANT_ID":              tenantID,
		"ARM_CLIENT_ID":              clientID,
		"ARM_USE_OIDC":               "true",
		"ARM_OIDC_TOKEN":             string(token),
		"AZURE_CLIENT_ID":            clientID,
		"AZURE_TENANT_ID":            tenantID,
		"AZURE_FEDERATED_TOKEN_FILE": filePath,
	}, env)
}
