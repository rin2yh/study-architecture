package main

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	t.Run("異常系 REDIS_URL 未指定で di.InitConsumer が失敗し起動前に error", func(t *testing.T) {
		t.Setenv("REDIS_URL", "")
		if err := run(context.Background()); err == nil {
			t.Fatal("run(): want error")
		}
	})

	t.Run("正常系 ctx キャンセル済みなら Run が context.Canceled を返し nil", func(t *testing.T) {
		// pgxpool.New / redis.ParseURL は遅延接続なので到達不能 URL でも InitConsumer は成功する。
		t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
		t.Setenv("REDIS_URL", "redis://127.0.0.1:1")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := run(ctx); err != nil {
			t.Fatalf("run() = %v, want nil", err)
		}
	})
}
