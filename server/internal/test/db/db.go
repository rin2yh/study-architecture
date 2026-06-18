// Package db は repository / handler の実 DB 結合テスト用ヘルパー。
package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SkipShort は -short 実行時に結合テストを skip する。DB を持たない per-service の単体ジョブは
// -short で回るため、各結合テストの先頭で呼んで skip させる (skip 判定を Open に持たせない)。
func SkipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skip integration test in -short mode")
	}
}

// Open は envVar が指す実 DB へ接続する。envVar は必須の契約で、未設定なら接続に失敗して
// Fatal になる (skip はしない)。-short での skip は SkipShort で別に行う。
func Open(t *testing.T, envVar string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), os.Getenv(envVar))
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
