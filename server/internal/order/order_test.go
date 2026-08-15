package order_test

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/order"
)

func TestParse(t *testing.T) {
	type want struct {
		id      string
		wantErr bool
	}
	tests := []struct {
		name string
		raw  string
		want want
	}{
		{"正常系 数値文字列から作る", "20", want{"20", false}},
		{"準正常系 パース不能な値は error", "abc", want{"0", true}},
		{"準正常系 空文字も error", "", want{"0", true}},
		{"準正常系 0 は採番されないので弾く", "0", want{"0", true}},
		{"準正常系 負値も弾く", "-1", want{"0", true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := order.Parse(tt.raw)
			if (err != nil) != tt.want.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if got.String() != tt.want.id {
				t.Fatalf("Parse() = %s, want %s", got.String(), tt.want.id)
			}
		})
	}
}

func TestParseIDFromEvent(t *testing.T) {
	type want struct {
		id      string
		wantErr bool
	}
	tests := []struct {
		name   string
		values map[string]any
		want   want
	}{
		{"正常系 イベントの values から復元する", map[string]any{order.FieldID: "20"}, want{"20", false}},
		{"準正常系 パース不能な値は error", map[string]any{order.FieldID: "abc"}, want{"0", true}},
		{"準正常系 フィールドが無い payload も error", map[string]any{}, want{"0", true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := order.ParseIDFromEvent(tt.values)
			if (err != nil) != tt.want.wantErr {
				t.Fatalf("ParseIDFromEvent() error = %v, wantErr %v", err, tt.want.wantErr)
			}
			if got.String() != tt.want.id {
				t.Fatalf("ParseIDFromEvent() = %s, want %s", got.String(), tt.want.id)
			}
		})
	}
}
