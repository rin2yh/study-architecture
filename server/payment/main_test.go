package main

import (
	"testing"
)

func TestHttpAddr(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	if got := httpAddr(); got != ":8080" {
		t.Fatalf("httpAddr() = %q, want :8080", got)
	}

	t.Setenv("HTTP_ADDR", ":9999")
	if got := httpAddr(); got != ":9999" {
		t.Fatalf("httpAddr() = %q, want :9999", got)
	}
}

func TestRunInitError(t *testing.T) {
	// DATABASE_URL を空にすると di.InitHandler→NewPool がエラーを返し、
	// run() はサーバ起動前に error を返す。
	t.Setenv("DATABASE_URL", "")
	if err := run(); err == nil {
		t.Fatal("run(): want error when DATABASE_URL is empty")
	}
}
