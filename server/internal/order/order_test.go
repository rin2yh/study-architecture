package order_test

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/order"
)

func TestIDFrom(t *testing.T) {
	type want struct {
		id      int64
		wantErr bool
	}
	tests := []struct {
		name   string
		values map[string]any
		want   want
	}{
		{"正常系 数値文字列を数値へ戻す", map[string]any{order.FieldID: "20"}, want{20, false}},
		{"準正常系 パース不能な値は error にして DLQ へ委ねる", map[string]any{order.FieldID: "abc"}, want{0, true}},
		{"準正常系 フィールドが無い payload も error", map[string]any{}, want{0, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := order.IDFrom(tt.values)
			if (err != nil) != tt.want.wantErr {
				t.Fatalf("IDFrom() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if got != tt.want.id {
				t.Fatalf("IDFrom() = %d, want %d", got, tt.want.id)
			}
		})
	}
}
