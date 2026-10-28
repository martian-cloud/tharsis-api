package resolver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	testActorPrincipalID = "5c9d0e1f-2a3b-4c5d-6e7f-8a9b0c1d2e3f"
	testActorMissingID   = "6d0e1f2a-3b4c-5d6e-7f8a-9b0c1d2e3f4a"
)

// The fakes embed the service interfaces and implement only the batch method the loaders call.
// Each returns the records whose IDs it knows, the way the services leave out records that were
// deleted or that the caller cannot view.

type fakeUserService struct {
	user.Service
	users []models.User
	err   error
}

func (f *fakeUserService) GetUsersByIDs(_ context.Context, _ []string) ([]models.User, error) {
	return f.users, f.err
}

type fakeServiceAccountService struct {
	serviceaccount.Service
	serviceAccounts []models.ServiceAccount
}

func (f *fakeServiceAccountService) GetServiceAccountsByIDs(_ context.Context, _ []string) ([]models.ServiceAccount, error) {
	return f.serviceAccounts, nil
}

type fakeJobService struct {
	job.Service
	jobs []models.Job
}

func (f *fakeJobService) GetJobsByIDs(_ context.Context, _ []string) ([]models.Job, error) {
	return f.jobs, nil
}

type fakeVCSService struct {
	vcs.Service
	providers []models.VCSProvider
}

func (f *fakeVCSService) GetVCSProvidersByIDs(_ context.Context, _ []string) ([]models.VCSProvider, error) {
	return f.providers, nil
}

func TestNewActorResolver(t *testing.T) {
	assert.Nil(t, newActorResolver(models.Actor{Subject: "legacy@example.com"}), "an untyped actor has no resolver")

	userActor := newActorResolver(models.NewUserActor(testActorPrincipalID).WithSubject("user@example.com"))
	require.NotNil(t, userActor)
	assert.Equal(t, "USER", userActor.Type())
	require.NotNil(t, userActor.PrincipalID())
	assert.Equal(t, gid.ToGlobalID(types.UserModelType, testActorPrincipalID), *userActor.PrincipalID())

	systemActor := newActorResolver(models.SystemActor())
	require.NotNil(t, systemActor)
	assert.Equal(t, "SYSTEM", systemActor.Type())
	assert.Nil(t, systemActor.PrincipalID(), "an actor without a backing record has no id")
}

// TestActorResolver_ConcreteType verifies that each actor resolves to exactly one implementing type,
// chosen from the stored actor type.
func TestActorResolver_ConcreteType(t *testing.T) {
	tests := []struct {
		name   string
		actor  models.Actor
		expect string
	}{
		{name: "user", actor: models.NewUserActor(testActorPrincipalID), expect: "UserActor"},
		{name: "service account", actor: models.NewServiceAccountActor(testActorPrincipalID), expect: "ServiceAccountActor"},
		{name: "vcs provider", actor: models.NewVCSProviderActor(testActorPrincipalID), expect: "VCSProviderActor"},
		{name: "job", actor: models.NewJobActor(testActorPrincipalID), expect: "JobActor"},
		{name: "system", actor: models.SystemActor(), expect: "SystemActor"},
		{name: "scim", actor: models.SCIMActor(), expect: "SCIMActor"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newActorResolver(test.actor)
			matched := map[string]bool{}
			_, matched["UserActor"] = r.ToUserActor()
			_, matched["ServiceAccountActor"] = r.ToServiceAccountActor()
			_, matched["VCSProviderActor"] = r.ToVCSProviderActor()
			_, matched["JobActor"] = r.ToJobActor()
			_, matched["SystemActor"] = r.ToSystemActor()
			_, matched["SCIMActor"] = r.ToSCIMActor()

			for typeName, ok := range matched {
				assert.Equal(t, typeName == test.expect, ok, typeName)
			}
		})
	}
}

// TestActorResolver_TRNSnapshotNeedsNoLookup verifies that an actor with a TRN snapshot is displayed from it
// alone: the context has no loaders attached, so any principal lookup would fail.
func TestActorResolver_TRNSnapshotNeedsNoLookup(t *testing.T) {
	serviceAccount, ok := newActorResolver(models.NewServiceAccountActor(testActorPrincipalID).WithTRN("trn:service_account:group/sa")).ToServiceAccountActor()
	require.True(t, ok)
	require.NotNil(t, serviceAccount.TRNSnapshot())
	assert.Equal(t, "trn:service_account:group/sa", *serviceAccount.TRNSnapshot())

	provider, ok := newActorResolver(models.NewVCSProviderActor(testActorPrincipalID).WithTRN("trn:vcs_provider:group/vcs")).ToVCSProviderActor()
	require.True(t, ok)
	require.NotNil(t, provider.TRNSnapshot())
	assert.Equal(t, "trn:vcs_provider:group/vcs", *provider.TRNSnapshot())

	job, ok := newActorResolver(models.NewJobActor(testActorPrincipalID).WithTRN("trn:job:group/ws/job-gid")).ToJobActor()
	require.True(t, ok)
	require.NotNil(t, job.TRNSnapshot())
	assert.Equal(t, "trn:job:group/ws/job-gid", *job.TRNSnapshot())

	withoutTRN, ok := newActorResolver(models.NewJobActor(testActorPrincipalID)).ToJobActor()
	require.True(t, ok)
	assert.Nil(t, withoutTRN.TRNSnapshot())
}

func TestActorResolver_Principals(t *testing.T) {
	catalog := &services.Catalog{
		UserService: &fakeUserService{users: []models.User{
			{Metadata: models.ResourceMetadata{ID: testActorPrincipalID}, Email: "user@example.com"},
		}},
		ServiceAccountService: &fakeServiceAccountService{serviceAccounts: []models.ServiceAccount{
			{Metadata: models.ResourceMetadata{ID: testActorPrincipalID}, Name: "sa"},
		}},
		JobService: &fakeJobService{jobs: []models.Job{
			{Metadata: models.ResourceMetadata{ID: testActorPrincipalID}},
		}},
		VCSService: &fakeVCSService{providers: []models.VCSProvider{
			{Metadata: models.ResourceMetadata{ID: testActorPrincipalID}, Name: "provider"},
		}},
	}

	// resolve returns the principal resolver for the actor, or nil when the principal is unavailable.
	resolve := func(t *testing.T, actor models.Actor) any {
		ctx := newActorTestContext(catalog)
		r := newActorResolver(actor)
		var (
			result any
			err    error
		)
		if a, ok := r.ToUserActor(); ok {
			var res *UserResolver
			if res, err = a.User(ctx); res != nil {
				result = res
			}
		} else if a, ok := r.ToServiceAccountActor(); ok {
			var res *ServiceAccountResolver
			if res, err = a.ServiceAccount(ctx); res != nil {
				result = res
			}
		} else if a, ok := r.ToVCSProviderActor(); ok {
			var res *VCSProviderResolver
			if res, err = a.VCSProvider(ctx); res != nil {
				result = res
			}
		} else if a, ok := r.ToJobActor(); ok {
			var res *JobResolver
			if res, err = a.Job(ctx); res != nil {
				result = res
			}
		}
		require.NoError(t, err)
		return result
	}

	user, ok := resolve(t, models.NewUserActor(testActorPrincipalID)).(*UserResolver)
	require.True(t, ok)
	assert.Equal(t, "user@example.com", user.user.Email)

	serviceAccount, ok := resolve(t, models.NewServiceAccountActor(testActorPrincipalID)).(*ServiceAccountResolver)
	require.True(t, ok)
	assert.Equal(t, "sa", serviceAccount.serviceAccount.Name)

	provider, ok := resolve(t, models.NewVCSProviderActor(testActorPrincipalID)).(*VCSProviderResolver)
	require.True(t, ok)
	assert.Equal(t, "provider", provider.vcsProvider.Name)

	job, ok := resolve(t, models.NewJobActor(testActorPrincipalID)).(*JobResolver)
	require.True(t, ok)
	assert.Equal(t, testActorPrincipalID, job.job.Metadata.ID)

	// A principal that was deleted or that the caller cannot view is left out of the batch, so it
	// resolves to null instead of failing the query.
	assert.Nil(t, resolve(t, models.NewUserActor(testActorMissingID)))
	assert.Nil(t, resolve(t, models.NewServiceAccountActor(testActorMissingID)))
	assert.Nil(t, resolve(t, models.NewVCSProviderActor(testActorMissingID)))
	assert.Nil(t, resolve(t, models.NewJobActor(testActorMissingID)))
}

// TestActorResolver_SharedBatch verifies that a principal missing from a batch does not affect
// another principal loaded in the same batch.
func TestActorResolver_SharedBatch(t *testing.T) {
	catalog := &services.Catalog{
		ServiceAccountService: &fakeServiceAccountService{serviceAccounts: []models.ServiceAccount{
			{Metadata: models.ResourceMetadata{ID: testActorPrincipalID}, Name: "sa"},
		}},
	}
	ctx := newActorTestContext(catalog)

	visible, _ := newActorResolver(models.NewServiceAccountActor(testActorPrincipalID)).ToServiceAccountActor()
	missing, _ := newActorResolver(models.NewServiceAccountActor(testActorMissingID)).ToServiceAccountActor()

	// Run both concurrently so the dataloader batches them into a single call.
	type outcome struct {
		serviceAccount *ServiceAccountResolver
		err            error
	}
	missingResult := make(chan outcome, 1)
	go func() {
		serviceAccount, err := missing.ServiceAccount(ctx)
		missingResult <- outcome{serviceAccount, err}
	}()

	serviceAccount, err := visible.ServiceAccount(ctx)
	require.NoError(t, err)
	require.NotNil(t, serviceAccount)
	assert.Equal(t, "sa", serviceAccount.serviceAccount.Name)

	other := <-missingResult
	require.NoError(t, other.err)
	assert.Nil(t, other.serviceAccount)
}

func TestActorResolver_PropagatesOtherErrors(t *testing.T) {
	catalog := &services.Catalog{
		UserService: &fakeUserService{err: errors.New("db down", errors.WithErrorCode(errors.EInternal))},
	}

	userActor, ok := newActorResolver(models.NewUserActor(testActorPrincipalID)).ToUserActor()
	require.True(t, ok)
	user, err := userActor.User(newActorTestContext(catalog))
	require.Error(t, err)
	assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	assert.Nil(t, user)
}

func newActorTestContext(catalog *services.Catalog) context.Context {
	collection := loader.NewCollection()
	RegisterUserLoader(collection)
	RegisterServiceAccountLoader(collection)
	RegisterJobLoader(collection)
	RegisterVCSProviderLoader(collection)

	ctx := (&State{ServiceCatalog: catalog}).Attach(context.Background())
	return collection.Attach(ctx)
}
