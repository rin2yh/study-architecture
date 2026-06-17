package main

import (
	"testing"
)

func TestHttpAddr(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{"empty env falls back to :8080", "", ":8080"},
		{"explicit env is used as-is", ":9999", ":9999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", tt.env)
			if got := httpAddr(); got != tt.want {
				t.Fatalf("httpAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunInitError(t *testing.T) {
	// DATABASE_URL を空にすると di.InitHandler→NewPool がエラーを返し、run() はサーバ起動前に error を返す。
	t.Setenv("DATABASE_URL", "")
	if err := run(); err == nil {
		t.Fatal("run(): want error when DATABASE_URL is empty")
	}
}
