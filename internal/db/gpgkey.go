package db

//go:generate go tool mockery --name GPGKeys --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

// GPGKeys encapsulates the logic to access gpg keys from the database
type GPGKeys interface {
	GetGPGKeyByID(ctx context.Context, id string) (*models.GPGKey, error)
	GetGPGKeyByTRN(ctx context.Context, trn string) (*models.GPGKey, error)
	GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*GPGKeysResult, error)
	CreateGPGKey(ctx context.Context, gpgKey *models.GPGKey) (*models.GPGKey, error)
	DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error
}

// GPGKeySortableField represents the fields that a gpg key can be sorted by
type GPGKeySortableField string

// GPGKeySortableField constants
const (
	GPGKeySortableFieldUpdatedAtAsc   GPGKeySortableField = "UPDATED_AT_ASC"
	GPGKeySortableFieldUpdatedAtDesc  GPGKeySortableField = "UPDATED_AT_DESC"
	GPGKeySortableFieldGroupLevelAsc  GPGKeySortableField = "GROUP_LEVEL_ASC"
	GPGKeySortableFieldGroupLevelDesc GPGKeySortableField = "GROUP_LEVEL_DESC"
)

func (sf GPGKeySortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case GPGKeySortableFieldUpdatedAtAsc, GPGKeySortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "gpg_keys", Col: "updated_at"}
	case GPGKeySortableFieldGroupLevelAsc, GPGKeySortableFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "group_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (sf GPGKeySortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (sf GPGKeySortableField) getTransformFunc() pagination.SortTransformFunc {
	switch sf {
	case GPGKeySortableFieldGroupLevelAsc, GPGKeySortableFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
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

func (t *terraformGPGKeys) GetGPGKeyByTRN(ctx context.Context, trn string) (*models.GPGKey, error) {
	ctx, span := tracer.Start(ctx, "db.GetGPGKeyByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	path, err := types.GPGKeyModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")

	if lastSlashIndex == -1 {
		return nil, errors.New("a GPG key TRN must have the group path and key fingerprint separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return t.getGPGKey(ctx, goqu.Ex{
		"gpg_keys.fingerprint": path[lastSlashIndex+1:],
		"namespaces.path":      path[:lastSlashIndex],
	})
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
	var sortTransformFunc pagination.SortTransformFunc
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
		sortTransformFunc = input.Sort.getTransformFunc()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "gpg_keys", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithSortByTransform(sortTransformFunc),
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
		item, err := scanGPGKey(rows)
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

	sql, args, err := dialect.From("gpg_keys").
		Prepared(true).
		With("gpg_keys",
			dialect.Insert("gpg_keys").
				Rows(
					goqu.Record{
						"id":          newResourceID(),
						"version":     initialResourceVersion,
						"created_at":  timestamp,
						"updated_at":  timestamp,
						"group_id":    gpgKey.GroupID,
						"gpg_key_id":  gpgKey.GPGKeyID,
						"fingerprint": gpgKey.Fingerprint,
						"ascii_armor": gpgKey.ASCIIArmor,
						"created_by":  gpgKey.CreatedBy,
					},
				).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"gpg_keys.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdKey, err := scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return createdKey, nil
}

func (t *terraformGPGKeys) DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error {
	ctx, span := tracer.Start(ctx, "db.DeleteGPGKey")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("gpg_keys").
		Prepared(true).
		With("gpg_keys",
			dialect.Delete("gpg_keys").
				Where(
					goqu.Ex{
						"id":      gpgKey.Metadata.ID,
						"version": gpgKey.Metadata.Version,
					},
				).Returning("*"),
		).Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"gpg_keys.group_id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
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
	ctx, span := tracer.Start(ctx, "db.getGPGKey")
	defer span.End()

	query := dialect.From(goqu.T("gpg_keys")).
		Prepared(true).
		Select(t.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"gpg_keys.group_id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	gpgKey, err := scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
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

func scanGPGKey(row scanner) (*models.GPGKey, error) {
	var groupPath string
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
		&groupPath,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	gpgKey.Metadata.TRN = types.GPGKeyModelType.BuildTRN(groupPath, gpgKey.Fingerprint)

	return gpgKey, nil
}
