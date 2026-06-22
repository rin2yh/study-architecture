package main

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	t.Run("異常系 DATABASE_URL 未指定で di.InitHandler が失敗し起動前に error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if err := run(context.Background()); err == nil {
			t.Fatal("run(): want error")
		}
	})

	t.Run("正常系 ctx キャンセル済みなら起動直後にグレースフル停止し nil", func(t *testing.T) {
		// pgxpool.New は遅延接続なので到達不能 DSN でも InitHandler は成功する。
		t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
		t.Setenv("LISTEN_ADDR", "127.0.0.1:0")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := run(ctx); err != nil {
			t.Fatalf("run() = %v, want nil", err)
		}
	})
}
