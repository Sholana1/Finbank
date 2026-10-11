package pgdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Conn interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

func (db *DB) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgdb: begin transaction: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgdb: commit transaction: %w", err)
	}

	return nil
}

func (db *DB) Conn(ctx context.Context) Conn {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.pool
}
