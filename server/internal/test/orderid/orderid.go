// Package orderid はテストから注文 ID を組み立てる手間を共有する。
package orderid

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/order"
)

// 失敗はテストの前提が崩れているだけなので、呼び出し側で扱わせない。
func Must(t *testing.T, raw string) order.ID {
	t.Helper()
	id, err := order.Parse(raw)
	if err != nil {
		t.Fatalf("order.Parse(%q): %v", raw, err)
	}
	return id
}
