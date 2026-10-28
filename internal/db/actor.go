package db

import (
	"database/sql"
	"slices"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// An actor is stored in four columns: the legacy created_by (or
// <prefix>_triggered_by) column holding Actor.Subject, and three nullable columns
// <prefix>_actor_type (VARCHAR), <prefix>_actor_id (UUID) and
// <prefix>_actor_trn (VARCHAR). The id column is NULL for actor types that
// reference no record (system, scim), and the trn column is NULL for actor
// types that carry no TRN (user, system, scim). All three are NULL when the
// actor has no type, which is the case for every row written before typed actors
// were introduced.
//
// Conversion is explicit rather than relying on sql.Scanner and driver.Valuer on
// models.Actor: goqu builds statements with Prepared(true) and pgx encodes the
// resulting args, and depending on driver interface handling for a column this
// widely used is not worth the risk. These helpers keep the behavior in one
// place and match the existing nullableString idiom.

// metadataAndCreatedByFieldList is metadataFieldList followed by the columns of
// the created_by actor. A table that records its creator starts its field list
// with it, so those columns always sit right after the metadata columns and each
// scan lists their targets in that place. Tables with other actors list those
// columns themselves.
//
// It is clipped so its length equals its capacity: every table appends its own
// columns to it, and each append must copy rather than write into a shared
// backing array.
var metadataAndCreatedByFieldList = slices.Clip(append(metadataFieldList,
	"created_by", "created_by_actor_type", "created_by_actor_id", "created_by_actor_trn"))

// actorColumns returns the values for an actor's <prefix>_actor_type,
// <prefix>_actor_id and <prefix>_actor_trn columns. All three are NULL for an
// untyped actor, the id is NULL for actor types that reference no record, and
// the trn is NULL when the actor has none.
//
// A typed actor that fails validation is an error rather than being written as
// NULL: it can only come from a bug in whatever built the actor, and failing the
// write surfaces that bug instead of silently losing the attribution.
func actorColumns(actor models.Actor) (sql.NullString, sql.NullString, sql.NullString, error) {
	if !actor.HasType() {
		return sql.NullString{}, sql.NullString{}, sql.NullString{}, nil
	}
	if err := actor.Validate(); err != nil {
		return sql.NullString{}, sql.NullString{}, sql.NullString{}, errors.Wrap(err, "invalid actor", errors.WithErrorCode(errors.EInternal))
	}

	actorType := sql.NullString{String: string(actor.Type), Valid: true}
	actorTRN := sql.NullString{String: actor.TRN, Valid: actor.TRN != ""}
	if !actor.Type.HasID() {
		return actorType, sql.NullString{}, actorTRN, nil
	}
	return actorType, sql.NullString{String: actor.ID, Valid: true}, actorTRN, nil
}

// optionalActorColumns returns the values for an optional actor's legacy subject
// column followed by its actorColumns. All four are NULL for a nil actor.
func optionalActorColumns(actor *models.Actor) (sql.NullString, sql.NullString, sql.NullString, sql.NullString, error) {
	if actor == nil {
		return sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, nil
	}
	actorType, actorID, actorTRN, err := actorColumns(*actor)
	return nullableString(actor.Subject), actorType, actorID, actorTRN, err
}

// scanActor converts the scanned subject, <prefix>_actor_type, <prefix>_actor_id
// and <prefix>_actor_trn columns into an actor; see models.ActorFromParts for
// how missing or malformed columns are read.
func scanActor(subjectCol, typeCol, idCol, trnCol sql.NullString) models.Actor {
	return models.ActorFromParts(subjectCol.String, typeCol.String, idCol.String, trnCol.String)
}

// scanOptionalActor is scanActor for an optional actor: it returns nil when all
// four columns are NULL.
func scanOptionalActor(subjectCol, typeCol, idCol, trnCol sql.NullString) *models.Actor {
	actor := scanActor(subjectCol, typeCol, idCol, trnCol)
	if actor.IsZero() {
		return nil
	}
	return &actor
}
