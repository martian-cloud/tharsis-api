package db

//go:generate go tool mockery --name ObjectStoreRefs --inpackage --case underscore

import (
	"context"
	"time"

	"github.com/doug-martin/goqu/v9"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	objectStoreRefClaimLeaseDuration = 5 * time.Minute
)

// RetainObjectRefFunc retains an uploaded object by linking it to its owning resource. Higher-level
// store methods return this callback; if it is never called, the janitor will eventually delete the
// object as orphaned.
type RetainObjectRefFunc func(ctx context.Context, ownerID string) error

// ObjectStoreRefOwner identifies the FK column a ref is linked to. Defined here so the column name
// stays a DB concern; callers pass a constant rather than a raw column string.
type ObjectStoreRefOwner string

// Owner constants -- one per FK column on object_store_refs.
const (
	ObjectStoreRefOwnerConfigurationVersion   ObjectStoreRefOwner = "configuration_version_id"
	ObjectStoreRefOwnerStateVersion           ObjectStoreRefOwner = "state_version_id"
	ObjectStoreRefOwnerRun                    ObjectStoreRefOwner = "run_id"
	ObjectStoreRefOwnerModuleVersion          ObjectStoreRefOwner = "module_version_id"
	ObjectStoreRefOwnerProviderVersion        ObjectStoreRefOwner = "provider_version_id"
	ObjectStoreRefOwnerProviderPlatform       ObjectStoreRefOwner = "provider_platform_id"
	ObjectStoreRefOwnerProviderMirrorPlatform ObjectStoreRefOwner = "provider_mirror_platform_id"
	ObjectStoreRefOwnerLogStream              ObjectStoreRefOwner = "log_stream_id"
	ObjectStoreRefOwnerLogStreamChunk         ObjectStoreRefOwner = "log_stream_chunk_id"
	ObjectStoreRefOwnerAgentSession           ObjectStoreRefOwner = "agent_session_id"
)

// CreateObjectStoreRefInput is the input for creating an object store reference. AvailableAt is
// optional; if nil it defaults to now. Callers provide only the object key; the owner FK is set
// separately via LinkRef once the owning resource has been written.
type CreateObjectStoreRefInput struct {
	AvailableAt *time.Time
	ObjectKey   string
}

// ObjectStoreRef represents a tracked object in the object store.
type ObjectStoreRef struct {
	ID         string
	ObjectKey  string
	ClaimCount int
}

// ObjectStoreRefs provides an interface for tracking and cleaning up object store objects.
type ObjectStoreRefs interface {
	CreateRef(ctx context.Context, input *CreateObjectStoreRefInput) error
	LinkRef(ctx context.Context, objectKey string, owner ObjectStoreRefOwner, ownerID string) error
	ClaimOrphanedRefs(ctx context.Context, limit uint) ([]ObjectStoreRef, error)
	DeleteRefs(ctx context.Context, ids []string) error
}

type objectStoreRefs struct {
	dbClient *Client
}

// NewObjectStoreRefs returns an ObjectStoreRefs instance.
func NewObjectStoreRefs(dbClient *Client) ObjectStoreRefs {
	return &objectStoreRefs{dbClient: dbClient}
}

// CreateRef inserts a new object store reference.
func (r *objectStoreRefs) CreateRef(ctx context.Context, input *CreateObjectStoreRefInput) error {
	ctx, span := tracer.Start(ctx, "db.CreateRef")
	defer span.End()

	availableAt := currentTime()
	if input.AvailableAt != nil {
		availableAt = *input.AvailableAt
	}

	sql, args, err := dialect.Insert("object_store_refs").
		Prepared(true).
		Rows(goqu.Record{
			"id":           newResourceID(),
			"created_at":   currentTime(),
			"available_at": availableAt,
			"object_key":   input.ObjectKey,
		}).
		// OnConflict: intentional convention exception — re-uploads to the same key must not fail; refreshing available_at preserves the existing FK.
		OnConflict(goqu.DoUpdate("object_key", goqu.Record{
			"available_at": goqu.L("EXCLUDED.available_at"),
		})).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	if _, err = r.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

// LinkRef sets the owner FK on a pending ref once its owning resource has been written. Every tracked
// upload creates an unlinked pending ref; the service links it here, ideally in the same transaction
// as the resource write so the two commit atomically (an unlinked ref is reclaimed after its grace period).
func (r *objectStoreRefs) LinkRef(ctx context.Context, objectKey string, owner ObjectStoreRefOwner, ownerID string) error {
	ctx, span := tracer.Start(ctx, "db.LinkRef")
	defer span.End()

	sql, args, err := dialect.Update("object_store_refs").
		Prepared(true).
		Set(goqu.Record{string(owner): ownerID}).
		Where(goqu.C("object_key").Eq(objectKey)).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	result, err := r.dbClient.getConnection(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if result.RowsAffected() == 0 {
		return errors.New("object store ref not found for key %q", objectKey,
			errors.WithErrorCode(errors.ENotFound),
			errors.WithSpan(span),
		)
	}

	return nil
}

// ClaimOrphanedRefs atomically claims up to limit refs whose owner FK has been nullified by a
// cascade delete. Claimed refs are leased for objectStoreRefClaimLeaseDuration.
func (r *objectStoreRefs) ClaimOrphanedRefs(ctx context.Context, limit uint) ([]ObjectStoreRef, error) {
	ctx, span := tracer.Start(ctx, "db.ClaimOrphanedRefs")
	defer span.End()

	if limit == 0 {
		return nil, nil
	}

	now := currentTime()

	// When adding a new tracked resource type, add its FK column here AND to the partial index WHERE
	// clause in the migration -- they must stay in sync or the janitor claims live objects of that type.
	orphaned := goqu.And(
		goqu.I("object_store_refs.run_id").IsNull(),
		goqu.I("object_store_refs.state_version_id").IsNull(),
		goqu.I("object_store_refs.configuration_version_id").IsNull(),
		goqu.I("object_store_refs.log_stream_id").IsNull(),
		goqu.I("object_store_refs.log_stream_chunk_id").IsNull(),
		goqu.I("object_store_refs.module_version_id").IsNull(),
		goqu.I("object_store_refs.provider_version_id").IsNull(),
		goqu.I("object_store_refs.provider_platform_id").IsNull(),
		goqu.I("object_store_refs.provider_mirror_platform_id").IsNull(),
		goqu.I("object_store_refs.agent_session_id").IsNull(),
		goqu.I("object_store_refs.available_at").Lte(now),
	)

	claimable := dialect.From("object_store_refs").
		Select(goqu.I("object_store_refs.id")).
		Where(orphaned).
		Order(goqu.I("object_store_refs.created_at").Asc()).
		Limit(limit).
		ForUpdate(goqu.SkipLocked)

	sql, args, err := dialect.Update(goqu.T("object_store_refs")).
		Prepared(true).
		With("claimable", claimable).
		Set(goqu.Record{
			"available_at": now.Add(objectStoreRefClaimLeaseDuration),
			"claim_count":  goqu.L("claim_count + 1"),
		}).
		From(goqu.T("claimable")).
		Where(goqu.I("object_store_refs.id").Eq(goqu.I("claimable.id"))).
		Returning(
			goqu.I("object_store_refs.id"),
			goqu.I("object_store_refs.object_key"),
			goqu.I("object_store_refs.claim_count"),
		).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := r.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	var refs []ObjectStoreRef
	for rows.Next() {
		var ref ObjectStoreRef
		if err := rows.Scan(&ref.ID, &ref.ObjectKey, &ref.ClaimCount); err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}
		refs = append(refs, ref)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	return refs, nil
}

// DeleteRefs removes a batch of refs after their objects have been successfully deleted.
func (r *objectStoreRefs) DeleteRefs(ctx context.Context, ids []string) error {
	ctx, span := tracer.Start(ctx, "db.DeleteRefs")
	defer span.End()

	sql, args, err := dialect.Delete("object_store_refs").
		Prepared(true).
		Where(goqu.C("id").In(ids)).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	if _, err = r.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}
