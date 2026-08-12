// Package order は order ドメインの識別子を、order 以外のサービスとも共有する形で定める。
// 注文 ID は payment / shipping / inventory のイベントにも載るが、定義するのは order 側。
package order

import (
	"fmt"
	"strconv"
)

const FieldID = "orderId"

// 検証を通らない値の混入を型で禁じるため。
type ID struct{ v int64 }

// 採番は 1 始まりなので、0 以下は「未設定のまま渡された」ことを意味する。
func New(v int64) (ID, error) {
	if v <= 0 {
		return ID{}, fmt.Errorf("invalid orderId %d", v)
	}
	return ID{v: v}, nil
}

func Parse(raw string) (ID, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return ID{}, fmt.Errorf("invalid orderId %q: %w", raw, err)
	}
	return New(v)
}

// パース不能な payload は握り潰さず error にして DLQ へ委ねる (ADR-[[202607301418]])。
func ParseIDFromEvent(event map[string]any) (ID, error) {
	raw, _ := event[FieldID].(string)
	return Parse(raw)
}

func (id ID) Int64() int64 { return id.v }
