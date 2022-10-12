package db

//go:generate mockery --name Transactions --inpackage --case underscore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

// Transactions exposes DB transaction support
type Transactions interface {
	BeginTx(ctx context.Context) (context.Context, error)
	CommitTx(ctx context.Context) error
	RollbackTx(ctx context.Context) error
}

type transactions struct {
	dbClient *Client
}

// NewTransactions returns an instance of the Transactions interface
func NewTransactions(dbClient *Client) Transactions {
	return &transactions{dbClient: dbClient}
}

// BeginTx starts a new transaction which gets added to the returned context
func (t *transactions) BeginTx(ctx context.Context) (context.Context, error) {
	var tx pgx.Tx
	var err error

	parentTx, ok := ctx.Value(txKey).(pgx.Tx)
	if !ok {
		// Transaction doesn't exist yet so create a new one
		tx, err = t.dbClient.conn.Begin(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		// Parent transaction already exists so create a child transaction
		tx, err = parentTx.Begin(ctx)
		if err != nil {
			return nil, err
		}
	}

	return context.WithValue(ctx, txKey, tx), nil
}

// CommitTx commits the transaction that is on the current context
func (t *transactions) CommitTx(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	if !ok {
		return fmt.Errorf("transaction missing from context")
	}
	return tx.Commit(ctx)
}

// RollbackTx rolls back the transaction that is on the current context
func (t *transactions) RollbackTx(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	if !ok {
		return fmt.Errorf("transaction missing from context")
	}

	err := tx.Rollback(ctx)

	// Avoid throwing unnecessary errors in caller.
	if err == pgx.ErrTxClosed {
		return nil
	}

	return err
}
