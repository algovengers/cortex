package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Db struct {
	conn *pgxpool.Pool
}

func Init(ctx context.Context, dbUrl string) (*Db, error) {
	conn, err := pgxpool.New(ctx, dbUrl)

	if err != nil {
		return nil, err
	}

	return &Db{
		conn: conn,
	}, nil
}
