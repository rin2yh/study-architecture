package eventfield_test

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/eventfield"
)

func TestRequired(t *testing.T) {
	type want struct {
		value int64
		err   bool
	}
	tests := []struct {
		name   string
		values map[string]any
		want   want
	}{
		{"正常系 記録どおりの型なら取り出せる", map[string]any{"amountCents": int64(300)}, want{300, false}},
		{"準正常系 欠落はゼロ値へ化けず error になる", map[string]any{}, want{0, true}},
		{"準正常系 型違いはゼロ値へ化けず error になる", map[string]any{"amountCents": "300"}, want{0, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eventfield.Required[int64](tt.values, "amountCents")
			if (err != nil) != tt.want.err {
				t.Fatalf("Required() error = %v, want error %v", err, tt.want.err)
			}
			if got != tt.want.value {
				t.Fatalf("Required() = %d, want %d", got, tt.want.value)
			}
		})
	}
}

func TestOptional(t *testing.T) {
	type want struct {
		value int64
		err   bool
	}
	tests := []struct {
		name   string
		values map[string]any
		want   want
	}{
		{"正常系 記録どおりの型なら取り出せる", map[string]any{"amountCents": int64(300)}, want{300, false}},
		{"正常系 欠落はゼロ値で通す", map[string]any{}, want{0, false}},
		{"準正常系 型違いは契約違反なので error になる", map[string]any{"amountCents": "300"}, want{0, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eventfield.Optional[int64](tt.values, "amountCents")
			if (err != nil) != tt.want.err {
				t.Fatalf("Optional() error = %v, want error %v", err, tt.want.err)
			}
			if got != tt.want.value {
				t.Fatalf("Optional() = %d, want %d", got, tt.want.value)
			}
		})
	}
}
