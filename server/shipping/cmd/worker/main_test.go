package main

import (
	"context"
	"testing"
)

func TestStart(t *testing.T) {
	t.Run("正常系 ctx キャンセル済みなら Run が context.Canceled を返し exit 0", func(t *testing.T) {
		// pgxpool.New も AWS SDK のクライアント生成も接続を張らないので、到達不能な設定でも InitConsumer は成功する。
		t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
		t.Setenv("AWS_REGION", "ap-northeast-1")
		t.Setenv("AWS_ENDPOINT_URL", "http://127.0.0.1:1")
		t.Setenv("ORDER_API_URL", "http://127.0.0.1:1")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if code := start(ctx); code != 0 {
			t.Fatalf("start() = %d, want 0", code)
		}
	})

	// exit 1 の原因を DATABASE_URL に絞る。
	t.Run("異常系 DATABASE_URL 未指定で di.InitWorker が失敗し exit 1", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		t.Setenv("AWS_REGION", "ap-northeast-1")
		t.Setenv("AWS_ENDPOINT_URL", "http://127.0.0.1:1")
		t.Setenv("ORDER_API_URL", "http://127.0.0.1:1")
		if code := start(context.Background()); code != 1 {
			t.Fatalf("start() = %d, want 1", code)
		}
	})
}
