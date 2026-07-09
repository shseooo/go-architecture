// Package database provides the MySQL connection and a context-based transaction
// manager shared by every module. Because all modules run on one *sql.DB, a
// transaction started here propagates through the context into any module's
// repository — enabling atomic cross-module operations in the monolith.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// Open builds and verifies a MySQL connection from environment variables.
func Open() (*sql.DB, error) {
	val := url.Values{}
	val.Add("parseTime", "1")
	val.Add("loc", "Asia/Jakarta")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s",
		os.Getenv("DATABASE_USER"), os.Getenv("DATABASE_PASS"),
		os.Getenv("DATABASE_HOST"), os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_NAME"), val.Encode())

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

type ctxTxKey struct{}

// TxManager runs a function inside a single transaction, injecting the *sql.Tx
// into the context so repositories transparently join it.
type TxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

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

// TxFromContext returns the transaction bound to ctx, or nil if there is none.
// Repositories use it to pick between the ambient tx and the pool.
func TxFromContext(ctx context.Context) *sql.Tx {
	if tx, ok := ctx.Value(ctxTxKey{}).(*sql.Tx); ok {
		return tx
	}
	return nil
}
