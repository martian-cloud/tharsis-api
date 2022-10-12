package azurefederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientID := "client1"
	tenantID := "tenant1"

	logger, _ := logger.NewForTest()
	authenticator := New(logger)

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

	env, err := authenticator.Authenticate(ctx, &identity, token)
	if err != nil {
		t.Fatal(err)
	}

	// Verify local server is running
	req, err := http.NewRequest("GET", env["ARM_OIDC_REQUEST_URL"], nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))

	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	var resp tokenResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(token), resp.Value)

	assert.Equal(t, map[string]string{
		"ARM_TENANT_ID":          tenantID,
		"ARM_CLIENT_ID":          clientID,
		"ARM_USE_OIDC":           "true",
		"ARM_OIDC_REQUEST_TOKEN": string(token),
		"ARM_OIDC_REQUEST_URL":   env["ARM_OIDC_REQUEST_URL"],
	}, env)
}
