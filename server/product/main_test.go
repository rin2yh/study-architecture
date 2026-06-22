package main

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	t.Run("正常系 ctx キャンセル済みなら起動直後にグレースフル停止し nil", func(t *testing.T) {
		// pgxpool.New は遅延接続なので到達不能 DSN でも InitHandler は成功する。
		t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/db?sslmode=disable")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := run(ctx, "127.0.0.1:0"); err != nil {
			t.Fatalf("run() = %v, want nil", err)
		}
	})

	t.Run("異常系 DATABASE_URL 未指定で di.InitHandler が失敗し起動前に error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if err := run(context.Background(), ":0"); err == nil {
			t.Fatal("run(): want error")
		}
	})
}

func TestStart(t *testing.T) {
	t.Run("異常系 DATABASE_URL 未指定で run が失敗し exit code 1", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if code := start(); code != 1 {
			t.Fatalf("start() = %d, want 1", code)
		}
	})
}
