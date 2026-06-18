// Package testdb は repository / handler の実 DB 結合テスト用の接続ヘルパー。
//
// ビルドタグ (//go:build integration) は使わず、-short 実行時 (per-service の単体ジョブ)
// と DSN 未設定時に t.Skip する。これにより DB が無い環境でも `go test ./...` が通り、
// DB を渡したジョブ (server-integration / mise run test:integration) でだけ実行される。
package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(t *testing.T, envVar string) *pgxpool.Pool {
	t.Helper()
	pool := openOrSkip(t, envVar)
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// OpenClosed は接続後すぐ Close した pool を返す。クエリがエラーを伝播する異常系の検証用。
func OpenClosed(t *testing.T, envVar string) *pgxpool.Pool {
	t.Helper()
	pool := openOrSkip(t, envVar)
	pool.Close()
	return pool
}

func openOrSkip(t *testing.T, envVar string) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("skip integration test in -short mode")
	}
	dsn := os.Getenv(envVar)
	if dsn == "" {
		t.Skipf("skip integration test: %s is not set", envVar)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	return pool
}
