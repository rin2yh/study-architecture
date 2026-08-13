package strconvx_test

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/strconvx"
)

func TestMustInt64(t *testing.T) {
	type want struct {
		v         int64
		wantPanic bool
	}
	tests := []struct {
		name string
		raw  string
		want want
	}{
		{"正常系 数値文字列を返す", "20", want{20, false}},
		{"正常系 負値も通す", "-1", want{-1, false}},
		{"準正常系 数値でなければ panic", "abc", want{0, true}},
		{"準正常系 空文字も panic", "", want{0, true}},
		{"準正常系 int64 に収まらない値も panic", "9223372036854775808", want{0, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); (r != nil) != tt.want.wantPanic {
					t.Fatalf("panic = %v, wantPanic %v", r, tt.want.wantPanic)
				}
			}()
			if got := strconvx.MustInt64(tt.raw); got != tt.want.v {
				t.Fatalf("MustInt64() = %d, want %d", got, tt.want.v)
			}
		})
	}
}
