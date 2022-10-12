package awsfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	authenticator, _ := New()

	dataBuffer, err := json.Marshal(&awsfederated.Data{Role: "testrole"})
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

	// Because the temporary file path/name is generated inside the authenticator,
	// this has to query the returned environment variable to get to the file.
	filePath := env["AWS_WEB_IDENTITY_TOKEN_FILE"]

	data, _ := os.ReadFile(filePath)
	assert.Equal(t, []byte("tokendata"), data)

	assert.Equal(t, map[string]string{
		"AWS_ROLE_ARN":                "testrole",
		"AWS_WEB_IDENTITY_TOKEN_FILE": filePath,
	}, env)
}
