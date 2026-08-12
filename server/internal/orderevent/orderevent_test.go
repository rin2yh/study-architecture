package orderevent_test

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/orderevent"
)

func TestOrderID(t *testing.T) {
	type want struct {
		id      int64
		wantErr bool
	}
	tests := []struct {
		name   string
		values map[string]any
		want   want
	}{
		{"正常系 数値文字列を数値へ戻す", map[string]any{orderevent.FieldOrderID: "20"}, want{20, false}},
		{"準正常系 パース不能な値は error にして DLQ へ委ねる", map[string]any{orderevent.FieldOrderID: "abc"}, want{0, true}},
		{"準正常系 フィールドが無い payload も error", map[string]any{}, want{0, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orderevent.OrderID(tt.values)
			if (err != nil) != tt.want.wantErr {
				t.Fatalf("OrderID() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if got != tt.want.id {
				t.Fatalf("OrderID() = %d, want %d", got, tt.want.id)
			}
		})
	}
}
