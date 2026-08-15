// Package order は order ドメインの識別子を、order 以外のサービスとも共有する形で定める。
// 注文 ID は payment / shipping / inventory のイベントにも載るが、定義するのは order 側。
package order

import (
	"fmt"

	"github.com/rin2yh/study-architecture/server/internal/strconvx"
)

const FieldID = "orderId"

// 検証を通らない値の混入を型で禁じるため。
type ID struct{ v int64 }

// 採番は 1 始まりなので、0 以下は「未設定のまま渡された」ことを意味する。
func Parse(raw string) (ID, error) {
	v, err := strconvx.ParseInt64(raw)
	if err != nil {
		return ID{}, fmt.Errorf("invalid orderId %q: %w", raw, err)
	}
	if v <= 0 {
		return ID{}, fmt.Errorf("invalid orderId %d", v)
	}
	return ID{v: v}, nil
}

func ParseIDFromEvent(event map[string]any) (ID, error) {
	raw, _ := event[FieldID].(string)
	return Parse(raw)
}

func (id ID) String() string { return strconvx.FormatInt64(id.v) }
