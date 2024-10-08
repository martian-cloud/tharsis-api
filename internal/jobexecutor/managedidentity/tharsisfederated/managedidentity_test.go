package tharsisfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/tharsisfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serviceAccountPath := "service/account/path"

	authenticator := New()

	dataBuffer, err := json.Marshal(&tharsisfederated.Data{ServiceAccountPath: serviceAccountPath})
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

	assert.Equal(t, map[string]string{
		"THARSIS_SERVICE_ACCOUNT_PATH":  serviceAccountPath,
		"THARSIS_SERVICE_ACCOUNT_TOKEN": string(token),
	}, env)
}
