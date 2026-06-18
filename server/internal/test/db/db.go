// Package db は repository / handler の実 DB 結合テスト用ヘルパー。
package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open は envVar が指す実 DB へ接続する。未設定なら skip ではなく Fatal にする。黙って skip すると
// CI が緑でも実 DB を検証しない「偽の緑」になるため。空 DSN は pgxpool が libpq デフォルト
// (localhost 等) へ繋ぎにいき誤接続・不明瞭なエラーになるので、ここで明示的に弾く。
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
