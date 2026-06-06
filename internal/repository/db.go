package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("record not found")

func NewDB(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	conn, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
