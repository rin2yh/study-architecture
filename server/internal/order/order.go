// Package order は order ドメインの識別子を、order 以外のサービスとも共有する形で定める。
// 注文 ID は payment / shipping / inventory のイベントにも載るが、定義するのは order 側。
package order

import (
	"fmt"
	"strconv"
)

// FieldID は注文 ID を wire に載せるときのキー。
const FieldID = "orderId"

// IDFrom はイベントの values から注文 ID を数値へ戻す。パース不能な payload は握り潰さず
// error にして DLQ へ委ねる (ADR-[[202607301418]])。
func IDFrom(values map[string]any) (int64, error) {
	raw, _ := values[FieldID].(string)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid orderId %q: %w", raw, err)
	}
	return id, nil
}
