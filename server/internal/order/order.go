// Package order は order ドメインの識別子を、order 以外のサービスとも共有する形で定める。
// 注文 ID は payment / shipping / inventory のイベントにも載るが、定義するのは order 側。
package order

import (
	"fmt"
	"strconv"
)

// FieldID は注文 ID を wire に載せるときのキー。
const FieldID = "orderId"

// ID の値を未公開にするのは、検証を通らない値の混入を型で禁じるため。生成は New / Parse だけ。
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
func IDFrom(values map[string]any) (ID, error) {
	raw, _ := values[FieldID].(string)
	return Parse(raw)
}

// sqlc 生成コードは int64 を受けるので、その境界で取り出す。
func (id ID) Int64() int64 { return id.v }

func (id ID) String() string { return strconv.FormatInt(id.v, 10) }
