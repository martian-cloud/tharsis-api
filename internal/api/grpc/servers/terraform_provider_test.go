package servers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// providerListStub records the inputs of the provider version and platform queries. The provider registry
// service has no mockery mock, so it embeds the interface: any other method panics.
type providerListStub struct {
	providerregistry.Service
	versionsInput  *providerregistry.GetProviderVersionsInput
	platformsInput *providerregistry.GetProviderPlatformsInput
}

func (s *providerListStub) GetProviderVersions(_ context.Context, input *providerregistry.GetProviderVersionsInput) (*db.ProviderVersionsResult, error) {
	s.versionsInput = input
	return &db.ProviderVersionsResult{PageInfo: emptyPageInfo()}, nil
}

func (s *providerListStub) GetProviderPlatforms(_ context.Context, input *providerregistry.GetProviderPlatformsInput) (*db.ProviderPlatformsResult, error) {
	s.platformsInput = input
	return &db.ProviderPlatformsResult{PageInfo: emptyPageInfo()}, nil
}

func emptyPageInfo() *pagination.PageInfo {
	return &pagination.PageInfo{TotalCount: func(context.Context) (int32, error) { return 0, nil }}
}

func TestTerraformProviderServer_GetTerraformProviderVersionsFilters(t *testing.T) {
	svc := &providerListStub{}
	catalog := &services.Catalog{TerraformProviderRegistryService: svc}
	catalog.Init()
	ctx := auth.WithCaller(context.Background(), &auth.SystemCaller{})

	_, err := NewTerraformProviderServer(catalog).GetTerraformProviderVersions(ctx, &pb.GetTerraformProviderVersionsRequest{
		ProviderId:         gid.ToGlobalID(types.TerraformProviderModelType, testGroupID),
		ShaSumsUploaded:    new(true),
		ShaSumsSigUploaded: new(false),
	})
	require.NoError(t, err)

	require.NotNil(t, svc.versionsInput)
	assert.Equal(t, new(true), svc.versionsInput.SHASumsUploaded)
	assert.Equal(t, new(false), svc.versionsInput.SHASumsSignatureUploaded)
}

func TestTerraformProviderServer_GetTerraformProviderPlatformsFilters(t *testing.T) {
	svc := &providerListStub{}
	catalog := &services.Catalog{TerraformProviderRegistryService: svc}
	catalog.Init()
	ctx := auth.WithCaller(context.Background(), &auth.SystemCaller{})

	_, err := NewTerraformProviderServer(catalog).GetTerraformProviderPlatforms(ctx, &pb.GetTerraformProviderPlatformsRequest{
		OperatingSystem: new("linux"),
		Architecture:    new("arm64"),
	})
	require.NoError(t, err)

	require.NotNil(t, svc.platformsInput)
	assert.Equal(t, new("linux"), svc.platformsInput.OperatingSystem)
	assert.Equal(t, new("arm64"), svc.platformsInput.Architecture)
}
