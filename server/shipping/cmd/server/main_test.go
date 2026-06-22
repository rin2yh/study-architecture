package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	t.Run("異常系 DATABASE_URL 未指定で di.InitHandler が失敗し起動前に error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if err := run(); err == nil {
			t.Fatal("run(): want error")
		}
	})
}
