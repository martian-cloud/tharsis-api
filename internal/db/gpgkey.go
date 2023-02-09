package db

//go:generate mockery --name GPGKeys --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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

func (ts GPGKeySortableField) getFieldDescriptor() *fieldDescriptor {
	switch ts {
	case GPGKeySortableFieldUpdatedAtAsc, GPGKeySortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "gpg_keys", col: "updated_at"}
	default:
		return nil
	}
}

func (ts GPGKeySortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(ts), "_DESC") {
		return DescSort
	}
	return AscSort
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
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *GPGKeyFilter
}

// GPGKeysResult contains the response data and page information
type GPGKeysResult struct {
	PageInfo *PageInfo
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
	return t.getGPGKey(ctx, goqu.Ex{"gpg_keys.id": id})
}

func (t *terraformGPGKeys) GetGPGKeys(ctx context.Context, input *GetGPGKeysInput) (*GPGKeysResult, error) {
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

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "gpg_keys", col: "id"},
		sortBy,
		sortDirection,
		gpgKeyFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, t.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.GPGKey{}
	for rows.Next() {
		item, err := scanGPGKey(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := GPGKeysResult{
		PageInfo: rows.getPageInfo(),
		GPGKeys:  results,
	}

	return &result, nil
}

func (t *terraformGPGKeys) CreateGPGKey(ctx context.Context, gpgKey *models.GPGKey) (*models.GPGKey, error) {
	timestamp := currentTime()

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
		return nil, err
	}

	createdKey, err := scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(
					errors.EConflict,
					fmt.Sprintf("GPG key with key fingerprint %s already exists in group", gpgKey.Fingerprint),
				)
			}
		}
		return nil, err
	}

	return createdKey, nil
}

func (t *terraformGPGKeys) DeleteGPGKey(ctx context.Context, gpgKey *models.GPGKey) error {

	sql, args, err := dialect.Delete("gpg_keys").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      gpgKey.Metadata.ID,
				"version": gpgKey.Metadata.Version,
			},
		).Returning(gpgKeyFieldList...).ToSQL()
	if err != nil {
		return err
	}

	_, err = scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}
		return err
	}

	return nil
}

func (t *terraformGPGKeys) getGPGKey(ctx context.Context, exp goqu.Ex) (*models.GPGKey, error) {
	query := dialect.From(goqu.T("gpg_keys")).
		Prepared(true).
		Select(t.getSelectFields()...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	gpgKey, err := scanGPGKey(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return selectFields
}

func scanGPGKey(row scanner) (*models.GPGKey, error) {
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

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return gpgKey, nil
}

func gpgKeyFieldResolver(key string, model interface{}) (string, error) {
	gpgKey, ok := model.(*models.GPGKey)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected GPG Key type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &gpgKey.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
