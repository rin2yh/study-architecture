package strconvx_test

import (
	"math"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/strconvx"
)

func TestParseInt64(t *testing.T) {
	type want struct {
		v       int64
		wantErr bool
	}
	tests := []struct {
		name string
		raw  string
		want want
	}{
		{"正常系 数値文字列を返す", "20", want{20, false}},
		{"準正常系 数値でなければ error", "abc", want{0, true}},
		// ParseInt は範囲外で clamp した値を error と一緒に返す。
		{"準正常系 int64 に収まらない値も error", "9223372036854775808", want{math.MaxInt64, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strconvx.ParseInt64(tt.raw)
			if (err != nil) != tt.want.wantErr {
				t.Fatalf("ParseInt64() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if got != tt.want.v {
				t.Fatalf("ParseInt64() = %d, want %d", got, tt.want.v)
			}
		})
	}
}

func TestFormatInt64(t *testing.T) {
	tests := []struct {
		name string
		v    int64
		want string
	}{
		{"正常系 10 進の文字列にする", 20, "20"},
		{"正常系 負値も符号付きで返す", -1, "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strconvx.FormatInt64(tt.v); got != tt.want {
				t.Fatalf("FormatInt64() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMustParseInt64(t *testing.T) {
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
			if got := strconvx.MustParseInt64(tt.raw); got != tt.want.v {
				t.Fatalf("MustParseInt64() = %d, want %d", got, tt.want.v)
			}
		})
	}
}
