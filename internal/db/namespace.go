package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
)

type namespaceRow struct {
	id          string
	path        string
	groupID     string
	workspaceID string
	version     int
}

var namespaceFieldList = []interface{}{"id", "version", "path", "group_id", "workspace_id"}

func getNamespaceByGroupID(ctx context.Context, conn connection, groupID string) (*namespaceRow, error) {
	return getNamespace(ctx, conn, goqu.Ex{"group_id": groupID})
}

func getNamespaceByWorkspaceID(ctx context.Context, conn connection, workspaceID string) (*namespaceRow, error) {
	return getNamespace(ctx, conn, goqu.Ex{"workspace_id": workspaceID})
}

func getNamespaceByPath(ctx context.Context, conn connection, path string) (*namespaceRow, error) {
	return getNamespace(ctx, conn, goqu.Ex{"path": path})
}

func getNamespace(ctx context.Context, conn connection, ex goqu.Ex) (*namespaceRow, error) {
	sql, args, err := dialect.From("namespaces").
		Prepared(true).
		Select(namespaceFieldList...).
		Where(ex).
		ToSQL()
	if err != nil {
		return nil, err
	}

	namespace, err := scanNamespace(conn.QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return namespace, nil
}

func createNamespace(ctx context.Context, conn connection, namespace *namespaceRow) (*namespaceRow, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Insert("namespaces").
		Prepared(true).
		Rows(goqu.Record{
			"id":           newResourceID(),
			"version":      initialResourceVersion,
			"created_at":   timestamp,
			"updated_at":   timestamp,
			"path":         namespace.path,
			"group_id":     nullableString(namespace.groupID),
			"workspace_id": nullableString(namespace.workspaceID),
		}).
		Returning(namespaceFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdNamespace, err := scanNamespace(conn.QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, fmt.Sprintf("namespace %s already exists", namespace.path))
			}
		}
		return nil, err
	}

	return createdNamespace, nil
}

func scanNamespace(row scanner) (*namespaceRow, error) {
	var groupID sql.NullString
	var workspaceID sql.NullString

	namespace := &namespaceRow{}

	err := row.Scan(
		&namespace.id,
		&namespace.version,
		&namespace.path,
		&groupID,
		&workspaceID,
	)

	if err != nil {
		return nil, err
	}

	if groupID.Valid {
		namespace.groupID = groupID.String
	}

	if workspaceID.Valid {
		namespace.workspaceID = workspaceID.String
	}

	return namespace, nil
}
