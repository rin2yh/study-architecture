package rdb

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	// プール統計の計装失敗はサービス起動を妨げない (ADR-[[202606241356]])
	if err := otelpgx.RecordStats(pool); err != nil {
		slog.Warn("otelpgx record stats failed", "error", err)
	}
	return pool, nil
}
