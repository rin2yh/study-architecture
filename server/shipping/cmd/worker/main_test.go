package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	t.Run("異常系 REDIS_URL 未指定で di.InitConsumer が失敗し起動前に error", func(t *testing.T) {
		t.Setenv("REDIS_URL", "")
		if err := run(); err == nil {
			t.Fatal("run(): want error")
		}
	})
}
