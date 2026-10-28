package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

/* Actor resolvers */

// ActorResolver resolves the Actor interface. Its concrete type is chosen from the stored actor type
// alone, before any field is resolved, so a principal is looked up only when a query selects the
// principal field of the matching type.
type ActorResolver struct {
	actorFields
}

// newActorResolver returns a resolver for the actor, or nil when it has no type. An untyped actor
// is a row written before typed actors were introduced; its subject is exposed only through the
// separate string field it sits beside.
func newActorResolver(actor models.Actor) *ActorResolver {
	if !actor.HasType() {
		return nil
	}
	return &ActorResolver{actorFields{actor: actor}}
}

// newOptionalActorResolver is newActorResolver for an optional actor, returning nil when there is
// no actor.
func newOptionalActorResolver(actor *models.Actor) *ActorResolver {
	if actor == nil {
		return nil
	}
	return newActorResolver(*actor)
}

// Make sure these stay consistent with the types implementing Actor in actor.graphql:

// ToUserActor resolves user actors
func (r *ActorResolver) ToUserActor() (*UserActorResolver, bool) {
	if r.actor.Type != models.ActorTypeUser {
		return nil, false
	}
	return &UserActorResolver{r.actorFields}, true
}

// ToServiceAccountActor resolves service account actors
func (r *ActorResolver) ToServiceAccountActor() (*ServiceAccountActorResolver, bool) {
	if r.actor.Type != models.ActorTypeServiceAccount {
		return nil, false
	}
	return &ServiceAccountActorResolver{r.actorFields}, true
}

// ToVCSProviderActor resolves VCS provider actors
func (r *ActorResolver) ToVCSProviderActor() (*VCSProviderActorResolver, bool) {
	if r.actor.Type != models.ActorTypeVCSProvider {
		return nil, false
	}
	return &VCSProviderActorResolver{r.actorFields}, true
}

// ToJobActor resolves job actors
func (r *ActorResolver) ToJobActor() (*JobActorResolver, bool) {
	if r.actor.Type != models.ActorTypeJob {
		return nil, false
	}
	return &JobActorResolver{r.actorFields}, true
}

// ToSystemActor resolves system actors
func (r *ActorResolver) ToSystemActor() (*SystemActorResolver, bool) {
	if r.actor.Type != models.ActorTypeSystem {
		return nil, false
	}
	return &SystemActorResolver{r.actorFields}, true
}

// ToSCIMActor resolves SCIM actors
func (r *ActorResolver) ToSCIMActor() (*SCIMActorResolver, bool) {
	if r.actor.Type != models.ActorTypeSCIM {
		return nil, false
	}
	return &SCIMActorResolver{r.actorFields}, true
}

// actorFields resolves the fields every Actor type shares.
type actorFields struct {
	actor models.Actor
}

// Type resolver
func (r actorFields) Type() string {
	return toGraphqlEnum(string(r.actor.Type))
}

// PrincipalID resolver returns the global ID of the referenced record, or null for actor types that
// reference no record. It is deliberately not named id: Relay normalizes every object by its id, so
// an actor sharing its principal's id would collide with the principal's record in the store.
func (r actorFields) PrincipalID() *string {
	globalID := r.actor.GlobalID()
	if globalID == "" {
		return nil
	}
	return &globalID
}

// trnSnapshot returns the TRN recorded with the action, or null when the actor has none.
func (r actorFields) trnSnapshot() *string {
	if r.actor.TRN == "" {
		return nil
	}
	actorTRN := r.actor.TRN
	return &actorTRN
}

// unavailablePrincipal reports whether a principal load failed only because the record was deleted
// or the caller cannot view it. The batch loaders omit such records, so both surface as not found,
// and the principal field resolves to null instead of failing the rest of the query.
func unavailablePrincipal(err error) bool {
	code := errors.ErrorCode(err)
	return code == errors.ENotFound || code == errors.EForbidden
}

// UserActorResolver resolves the UserActor type.
type UserActorResolver struct {
	actorFields
}

// User resolver
func (r *UserActorResolver) User(ctx context.Context) (*UserResolver, error) {
	user, err := loadUser(ctx, r.actor.ID)
	if err != nil {
		if unavailablePrincipal(err) {
			return nil, nil
		}
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// ServiceAccountActorResolver resolves the ServiceAccountActor type.
type ServiceAccountActorResolver struct {
	actorFields
}

// TRNSnapshot resolver
func (r *ServiceAccountActorResolver) TRNSnapshot() *string {
	return r.trnSnapshot()
}

// ServiceAccount resolver
func (r *ServiceAccountActorResolver) ServiceAccount(ctx context.Context) (*ServiceAccountResolver, error) {
	serviceAccount, err := loadServiceAccount(ctx, r.actor.ID)
	if err != nil {
		if unavailablePrincipal(err) {
			return nil, nil
		}
		return nil, err
	}
	return &ServiceAccountResolver{serviceAccount: serviceAccount}, nil
}

// VCSProviderActorResolver resolves the VCSProviderActor type.
type VCSProviderActorResolver struct {
	actorFields
}

// TRNSnapshot resolver
func (r *VCSProviderActorResolver) TRNSnapshot() *string {
	return r.trnSnapshot()
}

// VCSProvider resolver
func (r *VCSProviderActorResolver) VCSProvider(ctx context.Context) (*VCSProviderResolver, error) {
	provider, err := loadVCSProvider(ctx, r.actor.ID)
	if err != nil {
		if unavailablePrincipal(err) {
			return nil, nil
		}
		return nil, err
	}
	return &VCSProviderResolver{vcsProvider: provider}, nil
}

// JobActorResolver resolves the JobActor type.
type JobActorResolver struct {
	actorFields
}

// TRNSnapshot resolver
func (r *JobActorResolver) TRNSnapshot() *string {
	return r.trnSnapshot()
}

// Job resolver
func (r *JobActorResolver) Job(ctx context.Context) (*JobResolver, error) {
	job, err := loadJob(ctx, r.actor.ID)
	if err != nil {
		if unavailablePrincipal(err) {
			return nil, nil
		}
		return nil, err
	}
	return &JobResolver{job: job}, nil
}

// SystemActorResolver resolves the SystemActor type.
type SystemActorResolver struct {
	actorFields
}

// SCIMActorResolver resolves the SCIMActor type.
type SCIMActorResolver struct {
	actorFields
}
