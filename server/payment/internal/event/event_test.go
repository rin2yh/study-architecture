package event_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/payment/internal/event"
)

func TestSettled(t *testing.T) {
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
			if got := event.Settled(tt.status); got != tt.want.settled {
				t.Fatalf("Settled(%q) = %v, want %v", tt.status, got, tt.want.settled)
			}
		})
	}
}

func TestNewRedisPublisher(t *testing.T) {
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
			_, err := event.NewRedisPublisher()
			if tt.want.err && err == nil {
				t.Fatal("NewRedisPublisher(): want error")
			}
			if !tt.want.err && err != nil {
				t.Fatalf("NewRedisPublisher() = %v, want nil", err)
			}
		})
	}
}

func TestRedisPublisherPublishPaymentSettled(t *testing.T) {
	mr := miniredis.RunT(t)
	t.Setenv("REDIS_URL", "redis://"+mr.Addr())

	p, err := event.NewRedisPublisher()
	if err != nil {
		t.Fatalf("NewRedisPublisher: %v", err)
	}
	if err := p.PublishPaymentSettled(t.Context(), event.PaymentSettled{PaymentID: 1, OrderID: 20, AmountCents: 2980}); err != nil {
		t.Fatalf("PublishPaymentSettled: %v", err)
	}

	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	entries, err := rc.XRange(t.Context(), "payment.events", "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("stream length = %d, want 1", len(entries))
	}
	v := entries[0].Values
	type want struct{ key, val string }
	for _, w := range []want{
		{"event", "payment.settled"},
		{"paymentId", "1"},
		{"orderId", "20"},
		{"amountCents", "2980"},
	} {
		if got, _ := v[w.key].(string); got != w.val {
			t.Fatalf("values[%q] = %q, want %q", w.key, got, w.val)
		}
	}
}
