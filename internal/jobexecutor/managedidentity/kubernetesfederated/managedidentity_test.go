package kubernetesfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/kubernetesfederated"

	"github.com/stretchr/testify/assert"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientID := "client1"
	authenticator, _ := New()
	defer func(authenticator *Authenticator, ctx context.Context) {
		_ = authenticator.Close(ctx)
	}(authenticator, ctx)

	dataBuffer, err := json.Marshal(&kubernetesfederated.Data{Audience: clientID})
	if err != nil {
		t.Fatal(err)
	}

	identity := &pb.ManagedIdentity{
		Metadata: &pb.ResourceMetadata{
			Id: "managedIdentity-1",
		},
		Data: base64.StdEncoding.EncodeToString(dataBuffer),
	}
	token := []byte("tokendata")

	response, err := authenticator.Authenticate(
		ctx,
		[]*pb.ManagedIdentity{identity},
		func(_ context.Context, _ *pb.ManagedIdentity) ([]byte, error) {
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
