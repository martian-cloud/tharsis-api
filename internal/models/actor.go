package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// ActorType identifies the kind of principal that performed an action.
//
// An actor is persisted in four columns: the legacy created_by (VARCHAR), plus
// created_by_actor_type (VARCHAR), created_by_actor_id (UUID) and
// created_by_actor_trn (VARCHAR):
//
//	type             id      trn
//	user             <uuid>  NULL
//	service_account  <uuid>  trn:service_account:<resource path>
//	vcs_provider     <uuid>  trn:vcs_provider:<resource path>
//	job              <uuid>  trn:job:<workspace path>/<job global id>
//	system           NULL    NULL
//	scim             NULL    NULL
//
// Types that reference a record carry the record's UUID in Actor.ID. The
// remaining two identify a principal that has no backing record, so their ID
// is always empty and the id column is NULL. Keeping the ID in a native UUID
// column makes lookups by principal cheap.
//
// # TRN
//
// Actor.TRN is a snapshot of the principal's TRN taken when the action was
// recorded, so a client can render the actor without looking up its principal.
// It is set only for actors that are not people. A user never has a TRN: a
// user's TRN names its username, which like its email identifies a person, and
// a copy stored on every row the user touched would outlive the user's
// deletion. Clients resolve a user actor's principal instead, which is the
// cheapest principal lookup. Like Subject, the TRN is a display value, not
// part of the principal's identity (that is ID), and goes stale when the
// principal is renamed or moved.
//
// A NULL type column means the row has no typed actor. That is the case for
// every row written before typed actors were introduced, and such rows read back
// as an Actor whose Type is empty and whose Subject holds the created_by value.
// There is deliberately no "legacy" or "unknown" ActorType: an empty Type carries
// that meaning, and it stays visible to "WHERE created_by_actor_type IS NULL" so
// the un-migrated remainder is countable. Use HasType to tell the two apart.
//
// # Subject and the created_by column
//
// Actor.Subject carries the free-form created_by value -- an email, a service
// account resource path, a job TRN, or the literal "system" -- so a single Actor
// is written once and fills all three columns, and no existing client of
// created_by is affected. Subject is not part of the principal's identity:
// String, GlobalID and principal comparisons ignore it, because it changes when
// a principal is renamed.
//
// Subject is transitional. Removing it requires, in order:
//
//  1. Backfilling created_by_actor_type/created_by_actor_id for historical rows. Done: the
//     backfill migration (since folded into the consolidated init migration) mapped each
//     subject to the principal it named, and gave a subject that named no current
//     principal the type its format implies with UnmappedActorID.
//  2. Migrating every consumer that reads createdBy: the web UI, the Go SDK, the
//     Terraform provider, the CLI, the gRPC converters, and the MCP tool structs
//     (whose JSON schema descriptions currently say "Email address of the user
//     who created this group").
//  3. Adding batch lookup RPCs (GetUsersByIDs and friends) if gRPC clients need
//     to render a principal's current identity. Deliberately not done now:
//     per-client call volume is bounded by the rate limiter, and createdBy
//     remains available as a zero-lookup fallback for the whole transition.
//  4. Dropping the created_by column, then Actor.Subject and WithSubject.
//
// Until step 4, an email address remains in created_by for any row a user
// created. An erasure request therefore needs created_by rewritten for that
// principal across every table with a created_by_actor_id column, which is
// the column that makes finding those rows practical.
type ActorType string

// ActorType values.
const (
	// ActorTypeUser is a Tharsis user, identified by the user's ID.
	ActorTypeUser ActorType = "user"
	// ActorTypeServiceAccount is a service account, identified by its ID.
	ActorTypeServiceAccount ActorType = "service_account"
	// ActorTypeJob is a job running in a workspace, identified by the job's ID.
	ActorTypeJob ActorType = "job"
	// ActorTypeVCSProvider is a VCS provider acting through a workspace link,
	// identified by the provider's ID.
	ActorTypeVCSProvider ActorType = "vcs_provider"
	// ActorTypeSystem is an internal system action. It has no backing record.
	ActorTypeSystem ActorType = "system"
	// ActorTypeSCIM is a SCIM provisioning action. It has no backing record.
	ActorTypeSCIM ActorType = "scim"
)

// UnmappedActorID is the ID of an actor recorded before typed actors were
// introduced whose principal could not be identified, typically because it had
// been deleted. The actor's type is still known from the legacy subject's format.
// It references no record, so the principal resolves to nothing, but it is a
// valid UUID and code that requires an ID keeps working. Every unmapped actor of
// a type shares this ID, which is itself the signal that it was unmapped.
const UnmappedActorID = "00000000-0000-0000-0000-000000000000"

// actorTypeModelTypes maps each ActorType that references a record to the model
// type of that record. Membership in this map is what makes an ActorType
// ID-bearing, so it is the single source of truth for that distinction.
var actorTypeModelTypes = map[ActorType]types.ModelType{
	ActorTypeUser:           types.UserModelType,
	ActorTypeServiceAccount: types.ServiceAccountModelType,
	ActorTypeJob:            types.JobModelType,
	ActorTypeVCSProvider:    types.VCSProviderModelType,
}

// actorTypesWithoutID holds the ActorTypes that identify a principal with no
// backing record. Their ID is always empty.
var actorTypesWithoutID = map[ActorType]struct{}{
	ActorTypeSystem: {},
	ActorTypeSCIM:   {},
}

// actorTypesWithTRN holds the ActorTypes that may carry a TRN. They are the
// ID-bearing types whose principal is not a person; see ActorType.
var actorTypesWithTRN = map[ActorType]struct{}{
	ActorTypeServiceAccount: {},
	ActorTypeJob:            {},
	ActorTypeVCSProvider:    {},
}

// Actor identifies the principal that performed an action. ID holds the
// principal's record ID for ID-bearing types and is empty for the rest. TRN
// holds a snapshot of the principal's TRN for actors that are not people, and Subject holds the
// legacy created_by value; see ActorType.
//
// Model fields hold an Actor value. The zero Actor means the action has no
// recorded principal, and HasType is false for it; Actor is comparable, so two
// actors can be compared with ==.
type Actor struct {
	Type ActorType
	ID   string
	// TRN is a snapshot of the TRN of a service account, VCS provider or job. It
	// is empty for every other type.
	TRN string
	// Subject is the free-form value persisted in the legacy created_by column.
	// It is transitional and is removed together with that column.
	Subject string
}

// NewUserActor returns an Actor for the user with the given ID.
func NewUserActor(id string) Actor {
	return Actor{Type: ActorTypeUser, ID: id}
}

// NewServiceAccountActor returns an Actor for the service account with the given ID.
func NewServiceAccountActor(id string) Actor {
	return Actor{Type: ActorTypeServiceAccount, ID: id}
}

// NewJobActor returns an Actor for the job with the given ID.
func NewJobActor(id string) Actor {
	return Actor{Type: ActorTypeJob, ID: id}
}

// NewVCSProviderActor returns an Actor for the VCS provider with the given ID.
func NewVCSProviderActor(id string) Actor {
	return Actor{Type: ActorTypeVCSProvider, ID: id}
}

// SystemActor returns the Actor for internal system actions. Its Subject is the
// fixed "system" value that auth.System also uses.
func SystemActor() Actor {
	return Actor{Type: ActorTypeSystem, Subject: "system"}
}

// SCIMActor returns the Actor for SCIM provisioning actions. Its Subject is the
// fixed "scim" value the SCIM caller reports.
func SCIMActor() Actor {
	return Actor{Type: ActorTypeSCIM, Subject: "scim"}
}

// WithSubject returns a copy of the actor with Subject set, so a constructor call can
// be chained: models.NewUserActor(id).WithSubject(email). It is transitional and
// is removed together with Actor.Subject.
func (a Actor) WithSubject(subject string) Actor {
	a.Subject = subject
	return a
}

// WithTRN returns a copy of the actor with TRN set, so a constructor call can be
// chained: models.NewServiceAccountActor(id).WithTRN(serviceAccountTRN).
func (a Actor) WithTRN(trn string) Actor {
	a.TRN = trn
	return a
}

// HasID reports whether this ActorType references a record and therefore
// carries an ID.
func (t ActorType) HasID() bool {
	_, ok := actorTypeModelTypes[t]
	return ok
}

// valid reports whether this is a known ActorType.
func (t ActorType) valid() bool {
	if t.HasID() {
		return true
	}
	_, ok := actorTypesWithoutID[t]
	return ok
}

// IsZero reports whether the actor records no principal at all: it has neither
// a type nor a legacy Subject.
func (a Actor) IsZero() bool {
	return a == Actor{}
}

// HasType reports whether the actor has a type. It is false for the zero actor
// and for a row written before typed actors were introduced, which carries only a
// Subject.
func (a Actor) HasType() bool {
	return a.Type != ""
}

// ActorSubjectPtr returns a pointer to a copy of an optional actor's Subject, or
// nil when there is no actor or it has no subject, for API fields that report an
// absent principal as null. It is transitional and is removed together with
// Actor.Subject.
func ActorSubjectPtr(actor *Actor) *string {
	if actor == nil || actor.Subject == "" {
		return nil
	}
	return new(actor.Subject)
}

// String returns the typed identity of the actor: "<type>:<id>" for ID-bearing
// types and the bare type name for the rest. An actor without a type returns an
// empty string. TRN and Subject are not included.
func (a Actor) String() string {
	switch {
	case a.Type == "":
		return ""
	case a.Type.HasID():
		return string(a.Type) + ":" + a.ID
	default:
		return string(a.Type)
	}
}

// ModelType returns the model type of the record this actor references, and
// false for actor types that reference no record.
func (a Actor) ModelType() (types.ModelType, bool) {
	mt, ok := actorTypeModelTypes[a.Type]
	return mt, ok
}

// GlobalID returns the GID of the record this actor references, or an empty
// string for actor types that reference no record. It performs no database
// access.
func (a Actor) GlobalID() string {
	mt, ok := a.ModelType()
	if !ok || a.ID == "" {
		return ""
	}
	return gid.ToGlobalID(mt, a.ID)
}

// Validate returns an error if the actor's type and ID are not a valid typed
// identity, or if it carries a TRN its type does not allow. An actor without a
// type is not valid; callers that accept legacy, untyped actors check HasType
// first. Subject is not validated.
func (a Actor) Validate() error {
	if !a.Type.valid() {
		return errors.New("invalid actor type %q", a.Type, errors.WithErrorCode(errors.EInvalid))
	}

	// Refusing the TRN here, rather than ignoring it, keeps a person's
	// identifier from ever being written to a TRN column.
	if _, ok := actorTypesWithTRN[a.Type]; !ok && a.TRN != "" {
		return errors.New("actor type %s must not have a TRN", a.Type, errors.WithErrorCode(errors.EInvalid))
	}

	if a.Type.HasID() {
		// The ID's format is not checked: the actor_id columns are UUIDs, so the
		// database rejects a malformed ID on write and never returns one on read.
		if a.ID == "" {
			return errors.New("actor type %s requires an ID", a.Type, errors.WithErrorCode(errors.EInvalid))
		}
		return nil
	}

	if a.ID != "" {
		return errors.New("actor type %s must not have an ID, got %q", a.Type, a.ID,
			errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// ActorFromParts builds an actor from its stored columns: the legacy created_by
// subject and the separate <prefix>_actor_type, <prefix>_actor_id and
// <prefix>_actor_trn columns.
//
// A type and ID that do not form a valid actor yield an actor with only Subject
// set, or the zero actor when the subject is empty too. That covers an unrecognized type,
// which keeps a reader that predates a newly added ActorType working, and a
// type/ID combination that is wrong for a recognized type, which the write path
// refuses to persist and so can only come from outside the application. Either
// way the row reads as though it has no typed actor, exactly like a row written
// before typed actors existed, rather than failing every query that touches it.
func ActorFromParts(subject, actorType, id, trn string) Actor {
	actor := Actor{Type: ActorType(actorType), ID: id, TRN: trn, Subject: subject}
	if actor.Validate() == nil {
		return actor
	}
	return Actor{Subject: subject}
}
