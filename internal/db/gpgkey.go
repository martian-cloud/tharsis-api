package db

//go:generate mockery --name GPGKeys --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GPGKeys encapsulates the logic to access gpg keys from the database
type GPGKeys interface {
	GetGPGKeyByID(ctx context.Context, id string) (*models.GPGKey, error)
	GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*GPGKeysResult, error)
	CreateGPGKey(ctx context.Context, gpgKey *models.GPGKey) (*models.GPGKey, error)
	DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error
}

// GPGKeySortableField represents the fields that a gpg key can be sorted by
type GPGKeySortableField string

// GPGKeySortableField constants
const (
	GPGKeySortableFieldUpdatedAtAsc  GPGKeySortableField = "UPDATED_AT_ASC"
	GPGKeySortableFieldUpdatedAtDesc GPGKeySortableField = "UPDATED_AT_DESC"
)

func (ts GPGKeySortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch ts {
	case GPGKeySortableFieldUpdatedAtAsc, GPGKeySortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "gpg_keys", Col: "updated_at"}
	default:
		return nil
	}
}

func (ts GPGKeySortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// GPGKeyFilter contains the supported fields for filtering GPGKey resources
type GPGKeyFilter struct {
	GPGKeyID       *uint64
	KeyIDs         []string
	NamespacePaths []string
}

// GetGPGKeysInput is the input for listing GPG keys
type GetGPGKeysInput struct {
	// Sort specifies the field to sort on and direction
	Sort *GPGKeySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *GPGKeyFilter
}

// GPGKeysResult contains the response data and page information
type GPGKeysResult struct {
	PageInfo *pagination.PageInfo
	GPGKeys  []models.GPGKey
}

type terraformGPGKeys struct {
	dbClient *Client
}

var gpgKeyFieldList = append(metadataFieldList, "group_id", "gpg_key_id", "fingerprint", "ascii_armor", "created_by")

// NewGPGKeys returns an instance of the GPGKeys interface
func NewGPGKeys(dbClient *Client) GPGKeys {
	return &terraformGPGKeys{dbClient: dbClient}
}

func (t *terraformGPGKeys) GetGPGKeyByID(ctx context.Context, id string) (*models.GPGKey, error) {
	ctx, span := tracer.Start(ctx, "db.GetGPGKeyByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return t.getGPGKey(ctx, goqu.Ex{"gpg_keys.id": id})
}

func (t *terraformGPGKeys) GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*GPGKeysResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetGPGKeys")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.GPGKeyID != nil {
			ex = ex.Append(goqu.I("gpg_keys.gpg_key_id").Eq(*input.Filter.GPGKeyID))
		}

		if input.Filter.KeyIDs != nil {
			ex = ex.Append(goqu.I("gpg_keys.id").In(input.Filter.KeyIDs))
		}

		if input.Filter.NamespacePaths != nil {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
		}
	}

	query := dialect.From(goqu.T("gpg_keys")).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"gpg_keys.group_id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "gpg_keys", Col: "id"},
		sortBy,
		sortDirection,
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, t.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.GPGKey{}
	for rows.Next() {
		item, err := scanGPGKey(rows, true)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	result := GPGKeysResult{
		PageInfo: rows.GetPageInfo(),
		GPGKeys:  results,
	}

	return &result, nil
}

func (t *terraformGPGKeys) CreateGPGKey(ctx context.Context, gpgKey *models.GPGKey) (*models.GPGKey, error) {
	ctx, span := tracer.Start(ctx, "db.CreateGPGKey")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	tx, err := t.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			t.dbClient.logger.Errorf("failed to rollback tx for CreateGPGKey: %v", txErr)
		}
	}()

	sql, args, err := dialect.Insert("gpg_keys").
		Prepared(true).
		Rows(goqu.Record{
			"id":          newResourceID(),
			"version":     initialResourceVersion,
			"created_at":  timestamp,
			"updated_at":  timestamp,
			"group_id":    gpgKey.GroupID,
			"gpg_key_id":  gpgKey.GPGKeyID,
			"fingerprint": gpgKey.Fingerprint,
			"ascii_armor": gpgKey.ASCIIArmor,
			"created_by":  gpgKey.CreatedBy,
		}).
		Returning(gpgKeyFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdKey, err := scanGPGKey(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"GPG key with key fingerprint %s already exists in group", gpgKey.Fingerprint)
				return nil, errors.New(
					"GPG key with key fingerprint %s already exists in group", gpgKey.Fingerprint,
					errors.WithErrorCode(errors.EConflict),
				)
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Lookup namespace for group
	namespace, err := getNamespaceByGroupID(ctx, tx, createdKey.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace by group ID")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	// The fingerprint is guaranteed to be unique.
	createdKey.ResourcePath = buildGPGKeyResourcePath(namespace.path, createdKey.Fingerprint)

	return createdKey, nil
}

func (t *terraformGPGKeys) DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error {
	ctx, span := tracer.Start(ctx, "db.DeleteGPGKey")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("gpg_keys").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      gpgKey.Metadata.ID,
				"version": gpgKey.Metadata.Version,
			},
		).Returning(gpgKeyFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (t *terraformGPGKeys) getGPGKey(ctx context.Context, exp goqu.Ex) (*models.GPGKey, error) {
	query := dialect.From(goqu.T("gpg_keys")).
		Prepared(true).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"gpg_keys.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	gpgKey, err := scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return gpgKey, nil
}

func (t *terraformGPGKeys) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range gpgKeyFieldList {
		selectFields = append(selectFields, fmt.Sprintf("gpg_keys.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func buildGPGKeyResourcePath(groupPath string, keyFingerprint string) string {
	return fmt.Sprintf("%s/%s", groupPath, keyFingerprint)
}

func scanGPGKey(row scanner, withResourcePath bool) (*models.GPGKey, error) {
	gpgKey := &models.GPGKey{}

	fields := []interface{}{
		&gpgKey.Metadata.ID,
		&gpgKey.Metadata.CreationTimestamp,
		&gpgKey.Metadata.LastUpdatedTimestamp,
		&gpgKey.Metadata.Version,
		&gpgKey.GroupID,
		&gpgKey.GPGKeyID,
		&gpgKey.Fingerprint,
		&gpgKey.ASCIIArmor,
		&gpgKey.CreatedBy,
	}
	var path string
	if withResourcePath {
		fields = append(fields, &path)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if withResourcePath {
		gpgKey.ResourcePath = buildGPGKeyResourcePath(path, gpgKey.Fingerprint)
	}

	return gpgKey, nil
}
