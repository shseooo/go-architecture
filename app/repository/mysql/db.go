package mysql

import (
	"context"
	"database/sql"
)

// Querier is the subset of *sql.DB / *sql.Tx that the repositories use. Letting
// every query go through a Querier is what allows a repository to run either
// standalone (on the pool) or inside a transaction, transparently.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type ctxTxKey struct{}

// TxManager runs a function inside a single database transaction. It injects the
// *sql.Tx into the context so that any repository called within fn uses the same
// transaction — without the repositories knowing whether they are transactional.
type TxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// Do begins a transaction, runs fn with a tx-carrying context, then commits — or
// rolls back if fn returns an error or panics.
func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	return fn(context.WithValue(ctx, ctxTxKey{}, tx))
}

// querier returns the transaction bound to ctx if present, otherwise the pool.
func querier(ctx context.Context, db *sql.DB) Querier {
	if tx, ok := ctx.Value(ctxTxKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}
