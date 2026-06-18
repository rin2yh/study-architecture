// Package db は repository / handler の実 DB 結合テスト用ヘルパー。
package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open は envVar の DSN で実 DB へ接続する。env が空なら Fatal にする — 黙って skip すると
// CI が偽の緑になり、空 DSN のままだと pgxpool が libpq デフォルト先へ誤接続するため。
func Open(t *testing.T, envVar string) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv(envVar)
	if dsn == "" {
		t.Fatalf("%s is required for integration tests", envVar)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping (%s): %v", envVar, err)
	}
	t.Cleanup(pool.Close)
	return pool
}
