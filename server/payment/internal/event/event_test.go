package event_test

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/payment/internal/event"
)

func TestIsSettled(t *testing.T) {
	type want struct{ settled bool }
	tests := []struct {
		name   string
		status string
		want   want
	}{
		{"正常系 paid は確定", "paid", want{true}},
		{"正常系 settled は確定", "settled", want{true}},
		{"正常系 captured は確定", "captured", want{true}},
		{"準正常系 pending は未確定", "pending", want{false}},
		{"準正常系 refunded は未確定", "refunded", want{false}},
		{"準正常系 空文字は未確定", "", want{false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := event.IsSettled(tt.status); got != tt.want.settled {
				t.Fatalf("IsSettled(%q) = %v, want %v", tt.status, got, tt.want.settled)
			}
		})
	}
}
