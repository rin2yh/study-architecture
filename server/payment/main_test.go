package main

import (
	"context"
	"testing"
)

func TestStart(t *testing.T) {
	t.Run("正常系 ctx キャンセル済みなら起動→グレースフル停止で exit 0", func(t *testing.T) {
		// pgxpool.New / redis.ParseURL は遅延接続なので到達不能 URL でも InitApp は成功する。
		t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
		t.Setenv("REDIS_URL", "redis://127.0.0.1:1")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if code := start(ctx, "127.0.0.1:0"); code != 0 {
			t.Fatalf("start() = %d, want 0", code)
		}
	})

	t.Run("異常系 DATABASE_URL 未指定で di.InitApp が失敗し exit 1", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("REDIS_URL", "redis://127.0.0.1:1")
		if code := start(context.Background(), ":0"); code != 1 {
			t.Fatalf("start() = %d, want 1", code)
		}
	})
}
