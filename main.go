package main

import (
	"context"
	"dockertest-sqlc-test-sample/db"

	"github.com/jackc/pgx/v4"
)

func IncrementUserAges(ctx context.Context, conn *pgx.Conn, q *db.Queries, id int32) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	qWithTx := q.WithTx(tx)
	u, err := qWithTx.GetUser(ctx, id)
	if err != nil {
		return err
	}
	err = qWithTx.UpdateUserAges(ctx, db.UpdateUserAgesParams{
		ID:  u.ID,
		Age: u.Age + 1,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
