package broker_test

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/broker"
)

func TestNewClient(t *testing.T) {
	type args struct{ url string }
	type want struct{ err bool }
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 有効な REDIS_URL で生成できる", args{"redis://localhost:6379"}, want{false}},
		{"異常系 REDIS_URL 未指定は error", args{""}, want{true}},
		{"異常系 不正な URL は error", args{"::not-a-url"}, want{true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REDIS_URL", tt.args.url)
			_, err := broker.NewClient()
			if tt.want.err && err == nil {
				t.Fatal("NewClient(): want error")
			}
			if !tt.want.err && err != nil {
				t.Fatalf("NewClient() = %v, want nil", err)
			}
		})
	}
}
