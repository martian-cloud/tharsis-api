package db

import (
	"context"
	"database/sql"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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
	ctx, span := tracer.Start(ctx, "db.getNamespaceByGroupID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return getNamespace(ctx, conn, goqu.Ex{"group_id": groupID})
}

func getNamespaceByWorkspaceID(ctx context.Context, conn connection, workspaceID string) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.getNamespaceByWorkspaceID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return getNamespace(ctx, conn, goqu.Ex{"workspace_id": workspaceID})
}

func getNamespaceByPath(ctx context.Context, conn connection, path string) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.getNamespaceByPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return getNamespace(ctx, conn, goqu.Ex{"path": path})
}

func getNamespace(ctx context.Context, conn connection, ex goqu.Ex) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.getNamespace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("namespaces").
		Prepared(true).
		Select(namespaceFieldList...).
		Where(ex).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	namespace, err := scanNamespace(conn.QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return namespace, nil
}

func createNamespace(ctx context.Context, conn connection, namespace *namespaceRow) (*namespaceRow, error) {
	ctx, span := tracer.Start(ctx, "db.createNamespace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

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
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdNamespace, err := scanNamespace(conn.QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "namespace %s already exists", namespace.path)
				return nil, errors.New("namespace %s already exists", namespace.path, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdNamespace, nil
}

// migrateNamespaces migrates all namespaces that either exactly match an old path or have the old path as a prefix.
func migrateNamespaces(ctx context.Context, conn connection, oldPath, newPath string) error {
	ctx, span := tracer.Start(ctx, "db.migrateNamespaces")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("namespaces").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				"path":       goqu.L("REGEXP_REPLACE(?, ?, ?)", goqu.C("path"), oldPath, newPath),
			},
		).Where(goqu.Or(
		goqu.I("path").Eq(oldPath),
		goqu.I("path").Like(oldPath+"/%"),
	)).Returning(namespaceFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = conn.Exec(ctx, sql, args...)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute DB query")
		return err
	}

	return nil
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
