package db

//go:generate go tool mockery --name AsymSigningKeys --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// AsymSigningKeys encapsulates the logic to access asymmetric signing keys from the database
type AsymSigningKeys interface {
	GetAsymSigningKeyByID(ctx context.Context, id string) (*models.AsymSigningKey, error)
	GetAsymSigningKeyByTRN(ctx context.Context, trn string) (*models.AsymSigningKey, error)
	GetAsymSigningKeys(ctx context.Context, input *GetAsymSigningKeysInput) (*AsymSigningKeysResult, error)
	CreateAsymSigningKey(ctx context.Context, asymSigningKey *models.AsymSigningKey) (*models.AsymSigningKey, error)
	UpdateAsymSigningKey(ctx context.Context, asymSigningKey *models.AsymSigningKey) (*models.AsymSigningKey, error)
	DeleteAsymSigningKey(ctx context.Context, asymSigningKey *models.AsymSigningKey) error
}

// AsymSigningKeySortableField represents the fields that asymmetric signing keys can be sorted by
type AsymSigningKeySortableField string

// AsymSigningKeySortableField constants
const (
	AsymSigningKeySortableFieldCreatedAtAsc  AsymSigningKeySortableField = "CREATED_AT_ASC"
	AsymSigningKeySortableFieldCreatedAtDesc AsymSigningKeySortableField = "CREATED_AT_DESC"
)

func (as AsymSigningKeySortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch as {
	case AsymSigningKeySortableFieldCreatedAtAsc, AsymSigningKeySortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "asym_signing_keys", Col: "created_at"}
	default:
		return nil
	}
}

func (as AsymSigningKeySortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(as), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// AsymSigningKeyFilter contains the supported fields for filtering AsymSigningKey resources
type AsymSigningKeyFilter struct {
	Status []models.AsymSigningKeyStatus
}

// GetAsymSigningKeysInput is the input for listing asymmetric signing keys
type GetAsymSigningKeysInput struct {
	// Sort specifies the field to sort on and direction
	Sort *AsymSigningKeySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *AsymSigningKeyFilter
}

// AsymSigningKeysResult contains the response data and page information
type AsymSigningKeysResult struct {
	PageInfo        *pagination.PageInfo
	AsymSigningKeys []models.AsymSigningKey
}

type asymSigningKeys struct {
	dbClient *Client
}

var asymSigningKeysFieldList = append(metadataFieldList, "public_key", "pub_key_id", "plugin_data", "plugin_type", "status")

// NewAsymSigningKeys returns an instance of the AsymSigningKeys interface
func NewAsymSigningKeys(dbClient *Client) AsymSigningKeys {
	return &asymSigningKeys{dbClient: dbClient}
}

func (a *asymSigningKeys) GetAsymSigningKeyByID(ctx context.Context, id string) (*models.AsymSigningKey, error) {
	ctx, span := tracer.Start(ctx, "db.GetAsymSigningKeyByID")
	defer span.End()

	return a.getAsymSigningKey(ctx, goqu.Ex{"asym_signing_keys.id": id})
}

func (a *asymSigningKeys) GetAsymSigningKeyByTRN(ctx context.Context, trn string) (*models.AsymSigningKey, error) {
	ctx, span := tracer.Start(ctx, "db.GetAsymSigningKeyByTRN")
	defer span.End()

	path, err := types.AsymSigningKeyModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	return a.getAsymSigningKey(ctx, goqu.Ex{"asym_signing_keys.id": gid.FromGlobalID(path)})
}

func (a *asymSigningKeys) GetAsymSigningKeys(ctx context.Context, input *GetAsymSigningKeysInput) (*AsymSigningKeysResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetAsymSigningKeys")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.Status != nil && len(input.Filter.Status) > 0 {
			ex = ex.Append(goqu.I("asym_signing_keys.status").In(input.Filter.Status))
		}
	}

	query := dialect.From(goqu.T("asym_signing_keys")).
		Select(a.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "asym_signing_keys", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, a.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.AsymSigningKey{}
	for rows.Next() {
		item, aErr := scanAsymSigningKey(rows)
		if aErr != nil {
			return nil, errors.Wrap(aErr, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err = rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := AsymSigningKeysResult{
		PageInfo:        rows.GetPageInfo(),
		AsymSigningKeys: results,
	}

	return &result, nil
}

func (a *asymSigningKeys) CreateAsymSigningKey(ctx context.Context, asymSigningKey *models.AsymSigningKey) (*models.AsymSigningKey, error) {
	ctx, span := tracer.Start(ctx, "db.CreateAsymSigningKey")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("asym_signing_keys").
		Prepared(true).
		With("asym_signing_keys",
			dialect.Insert("asym_signing_keys").Rows(
				goqu.Record{
					"id":          newResourceID(),
					"version":     initialResourceVersion,
					"created_at":  timestamp,
					"updated_at":  timestamp,
					"public_key":  asymSigningKey.PublicKey,
					"pub_key_id":  asymSigningKey.PubKeyID,
					"plugin_data": asymSigningKey.PluginData,
					"plugin_type": asymSigningKey.PluginType,
					"status":      asymSigningKey.Status,
				}).Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdAsymSigningKey, err := scanAsymSigningKey(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdAsymSigningKey, nil
}

func (a *asymSigningKeys) UpdateAsymSigningKey(ctx context.Context, asymSigningKey *models.AsymSigningKey) (*models.AsymSigningKey, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateAsymSigningKey")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("asym_signing_keys").
		Prepared(true).
		With("asym_signing_keys",
			dialect.Update("asym_signing_keys").
				Set(goqu.Record{
					"version":     goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at":  timestamp,
					"public_key":  asymSigningKey.PublicKey,
					"pub_key_id":  asymSigningKey.PubKeyID,
					"plugin_data": asymSigningKey.PluginData,
					"plugin_type": asymSigningKey.PluginType,
					"status":      asymSigningKey.Status,
				}).Where(goqu.Ex{"id": asymSigningKey.Metadata.ID, "version": asymSigningKey.Metadata.Version}).
				Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedAsymSigningKey, err := scanAsymSigningKey(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedAsymSigningKey, nil
}

func (a *asymSigningKeys) DeleteAsymSigningKey(ctx context.Context, asymSigningKey *models.AsymSigningKey) error {
	ctx, span := tracer.Start(ctx, "db.DeleteAsymSigningKey")
	defer span.End()

	sql, args, err := dialect.From("asym_signing_keys").
		Prepared(true).
		With("asym_signing_keys",
			dialect.Delete("asym_signing_keys").
				Where(goqu.Ex{"id": asymSigningKey.Metadata.ID, "version": asymSigningKey.Metadata.Version}).
				Returning("*"),
		).Select(a.getSelectFields()...).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	_, err = scanAsymSigningKey(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (a *asymSigningKeys) getAsymSigningKey(ctx context.Context, exp goqu.Ex) (*models.AsymSigningKey, error) {
	ctx, span := tracer.Start(ctx, "db.getAsymSigningKey")
	defer span.End()

	query := dialect.From(goqu.T("asym_signing_keys")).
		Prepared(true).
		Select(a.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	asymSigningKey, err := scanAsymSigningKey(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return asymSigningKey, nil
}

func (*asymSigningKeys) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range asymSigningKeysFieldList {
		selectFields = append(selectFields, fmt.Sprintf("asym_signing_keys.%s", field))
	}

	return selectFields
}

func scanAsymSigningKey(row scanner) (*models.AsymSigningKey, error) {
	asymSigningKey := &models.AsymSigningKey{}

	fields := []interface{}{
		&asymSigningKey.Metadata.ID,
		&asymSigningKey.Metadata.CreationTimestamp,
		&asymSigningKey.Metadata.LastUpdatedTimestamp,
		&asymSigningKey.Metadata.Version,
		&asymSigningKey.PublicKey,
		&asymSigningKey.PubKeyID,
		&asymSigningKey.PluginData,
		&asymSigningKey.PluginType,
		&asymSigningKey.Status,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	asymSigningKey.Metadata.TRN = types.AsymSigningKeyModelType.BuildTRN(asymSigningKey.GetGlobalID())

	return asymSigningKey, nil
}
