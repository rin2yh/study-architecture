package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		want        bool
	}{
		{"異常系 DATABASE_URL 未指定で di.InitHandler が失敗し起動前に error", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.databaseURL)
			if err := run(); tt.want != (err != nil) {
				t.Fatalf("run() error = %v, want error: %v", err, tt.want)
			}
		})
	}
}
