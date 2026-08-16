package sqsx_test

import (
	"os"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/sqsx"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
)

// ビルドタグを使わず環境変数で分ける方針は ADR-[[202606180902]] と同じ。
func newClient(t *testing.T) *sqsx.Client {
	t.Helper()
	skip.Short(t)
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		t.Skip("AWS_ENDPOINT_URL is required for broker integration tests")
	}
	c, err := sqsx.NewClient(t.Context())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestPublishAndReceive(t *testing.T) {
	c := newClient(t)
	const topic, queue = "sqsx-test-events", "sqsx-test-events-consumer"

	sub, err := c.Subscribe(t.Context(), topic, queue)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if err := c.Publish(t.Context(), topic, map[string]any{"event": "test.happened", "orderId": "20", "amountCents": int64(1980)}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	msgs, err := sub.Receive(t.Context())
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	// 互換実装は購読が重複しうるので、件数でなく中身で見る。
	if len(msgs) == 0 {
		t.Fatal("received no messages")
	}
	if got, _ := msgs[0].Values["orderId"].(string); got != "20" {
		t.Fatalf("values[orderId] = %v, want \"20\"", msgs[0].Values["orderId"])
	}
	if got, _ := msgs[0].Values["amountCents"].(int64); got != 1980 {
		t.Fatalf("values[amountCents] = %v, want 1980", msgs[0].Values["amountCents"])
	}

	if err := sub.Ack(t.Context(), msgs[0].Handle); err != nil {
		t.Fatalf("Ack: %v", err)
	}
}
