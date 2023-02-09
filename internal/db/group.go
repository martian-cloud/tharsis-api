package db

//go:generate mockery --name Groups --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Groups encapsulates the logic to access groups from the database
type Groups interface {
	// GetGroupByID returns a group by ID
	GetGroupByID(ctx context.Context, id string) (*models.Group, error)
	// GetGroupByFullPath returns a group by full path
	GetGroupByFullPath(ctx context.Context, path string) (*models.Group, error)
	// DeleteGroup deletes a group
	DeleteGroup(ctx context.Context, group *models.Group) error
	// GetGroups returns a list of groups
	GetGroups(ctx context.Context, input *GetGroupsInput) (*GroupsResult, error)
	// CreateGroup creates a new group
	CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error)
	// UpdateGroup updates an existing group
	UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error)
}

// GroupFilter contains the supported fields for filtering Group resources
type GroupFilter struct {
	ParentID     *string
	GroupIDs     []string
	NamespaceIDs []string
	RootOnly     bool
}

// GroupSortableField represents the fields that a group can be sorted by
type GroupSortableField string

// GroupSortableField constants
const (
	GroupSortableFieldFullPathAsc  GroupSortableField = "FULL_PATH_ASC"
	GroupSortableFieldFullPathDesc GroupSortableField = "FULL_PATH_DESC"
)

func (gs GroupSortableField) getFieldDescriptor() *fieldDescriptor {
	switch gs {
	case GroupSortableFieldFullPathAsc, GroupSortableFieldFullPathDesc:
		return &fieldDescriptor{key: "full_path", table: "namespaces", col: "path"}
	default:
		return nil
	}
}

func (gs GroupSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(gs), "_DESC") {
		return DescSort
	}
	return AscSort
}

// GetGroupsInput is the input for listing groups
type GetGroupsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *GroupSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *GroupFilter
}

// GroupsResult contains the response data and page information
type GroupsResult struct {
	PageInfo *PageInfo
	Groups   []models.Group
}

var groupFieldList = append(metadataFieldList, "name", "description", "parent_id", "created_by")

type groups struct {
	dbClient *Client
}

// NewGroups returns an instance of the Groups interface
func NewGroups(dbClient *Client) Groups {
	return &groups{dbClient: dbClient}
}

func (g *groups) GetGroupByID(ctx context.Context, id string) (*models.Group, error) {
	return g.getGroup(ctx, goqu.Ex{"groups.id": id})
}

func (g *groups) GetGroupByFullPath(ctx context.Context, path string) (*models.Group, error) {
	return g.getGroup(ctx, goqu.Ex{"namespaces.path": path})
}

func (g *groups) GetGroups(ctx context.Context, input *GetGroupsInput) (*GroupsResult, error) {
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.RootOnly {
			ex["groups.parent_id"] = nil
		}

		if input.Filter.GroupIDs != nil {
			// This check avoids an SQL syntax error if an empty slice is provided.
			if len(input.Filter.GroupIDs) > 0 {
				ex["groups.id"] = input.Filter.GroupIDs
			}
		}

		if input.Filter.ParentID != nil {
			ex["groups.parent_id"] = *input.Filter.ParentID
		}

		if input.Filter.NamespaceIDs != nil {
			if len(input.Filter.NamespaceIDs) == 0 {
				return &GroupsResult{
					PageInfo: &PageInfo{},
					Groups:   []models.Group{},
				}, nil
			}

			ex["namespaces.id"] = input.Filter.NamespaceIDs
		}
	}

	query := dialect.From(goqu.T("groups")).
		Select(g.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"groups.id": goqu.I("namespaces.group_id")})).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "groups", col: "id"},
		sortBy,
		sortDirection,
		groupFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, g.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Group{}
	for rows.Next() {
		item, err := scanGroup(rows, true)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := GroupsResult{
		PageInfo: rows.getPageInfo(),
		Groups:   results,
	}

	return &result, nil
}

func (g *groups) CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	// Use transaction to update groups and namespaces tables
	tx, err := g.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			g.dbClient.logger.Errorf("failed to rollback tx for CreateGroup: %v", txErr)
		}
	}()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("groups").
		Prepared(true).
		Rows(goqu.Record{
			"id":          newResourceID(),
			"version":     initialResourceVersion,
			"created_at":  timestamp,
			"updated_at":  timestamp,
			"name":        group.Name,
			"description": nullableString(group.Description),
			"parent_id":   nullableString(group.ParentID),
			"created_by":  group.CreatedBy,
		}).
		Returning(groupFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdGroup, err := scanGroup(tx.QueryRow(ctx, sql, args...), false)

	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_parent_id" {
				return nil, errors.NewError(errors.EConflict, "invalid group parent: the specified parent group does not exist")
			}

			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, err
	}

	fullPath := group.Name

	// Lookup namespace for parent group if this is a nested group
	if group.ParentID != "" {
		parentNamespace, err := getNamespaceByGroupID(ctx, tx, group.ParentID)
		if err != nil {
			return nil, err
		}

		fullPath = fmt.Sprintf("%s/%s", parentNamespace.path, fullPath)
	}

	// Create new namespace resource for group
	if _, err := createNamespace(ctx, tx, &namespaceRow{path: fullPath, groupID: createdGroup.Metadata.ID}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	createdGroup.FullPath = fullPath

	return createdGroup, nil
}

func (g *groups) UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Update("groups").
		Prepared(true).
		Set(
			goqu.Record{
				"version":     goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":  timestamp,
				"description": nullableString(group.Description),
			},
		).Where(goqu.Ex{"id": group.Metadata.ID, "version": group.Metadata.Version}).Returning(groupFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedGroup, err := scanGroup(g.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	namespace, err := getNamespaceByGroupID(ctx, g.dbClient.getConnection(ctx), updatedGroup.Metadata.ID)
	if err != nil {
		return nil, err
	}

	updatedGroup.FullPath = namespace.path

	return updatedGroup, nil
}

func (g *groups) DeleteGroup(ctx context.Context, group *models.Group) error {
	sql, args, err := dialect.Delete("groups").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      group.Metadata.ID,
				"version": group.Metadata.Version,
			},
		).Returning(groupFieldList...).ToSQL()

	if err != nil {
		return err
	}

	if _, err := scanGroup(g.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_parent_id" {
				return errors.NewError(errors.EConflict, "all nested groups and workspaces must be deleted before this group can be deleted")
			}
		}

		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		return err
	}

	return nil
}

func (g *groups) getGroup(ctx context.Context, exp exp.Expression) (*models.Group, error) {
	query := dialect.From(goqu.T("groups")).
		Prepared(true).
		Select(g.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"groups.id": goqu.I("namespaces.group_id")})).Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	group, err := scanGroup(g.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return group, nil
}

func (g *groups) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range groupFieldList {
		selectFields = append(selectFields, fmt.Sprintf("groups.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanGroup(row scanner, withFullPath bool) (*models.Group, error) {
	var parentID sql.NullString
	var description sql.NullString
	var err error

	group := &models.Group{}

	fields := []interface{}{
		&group.Metadata.ID,
		&group.Metadata.CreationTimestamp,
		&group.Metadata.LastUpdatedTimestamp,
		&group.Metadata.Version,
		&group.Name,
		&description,
		&parentID,
		&group.CreatedBy,
	}

	if withFullPath {
		fields = append(fields, &group.FullPath)
	}

	err = row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		group.ParentID = parentID.String
	}

	if description.Valid {
		group.Description = description.String
	}

	return group, nil
}

func groupFieldResolver(key string, model interface{}) (string, error) {
	group, ok := model.(*models.Group)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected group type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &group.Metadata)
	if !ok {
		switch key {
		case "full_path":
			val = group.FullPath
		default:
			return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
		}
	}

	return val, nil
}
