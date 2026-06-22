package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		redisURL string
		want     bool
	}{
		{"異常系 REDIS_URL 未指定で di.InitConsumer が失敗し起動前に error", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REDIS_URL", tt.redisURL)
			if err := run(); tt.want != (err != nil) {
				t.Fatalf("run() error = %v, want error: %v", err, tt.want)
			}
		})
	}
}
