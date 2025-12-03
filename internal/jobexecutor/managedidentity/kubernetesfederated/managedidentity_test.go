package kubernetesfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/kubernetesfederated"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientID := "client1"
	authenticator, _ := New()
	defer func(authenticator *Authenticator, ctx context.Context) {
		err := authenticator.Close(ctx)
		if err != nil {

		}
	}(authenticator, ctx)

	dataBuffer, err := json.Marshal(&kubernetesfederated.Data{Audience: clientID})
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

	response, err := authenticator.Authenticate(
		ctx,
		[]types.ManagedIdentity{identity},
		func(_ context.Context, _ *types.ManagedIdentity) ([]byte, error) {
			return token, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, map[string]string{
		"KUBE_TOKEN": string(token),
	}, response.Env)

	assert.Nil(t, response.HostCredentialFileMapping)
}
