// Package strconvx は strconv の補助を共有層に置く。
package strconvx

import (
	"fmt"
	"strconv"
)

// MustInt64 は数値でない raw を panic で弾く。呼び出し側が扱えない失敗を error にしないための契約で、
// 検証済みの値しか渡らない箇所に限って使う。
func MustInt64(raw string) int64 {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("strconvx: invalid int64 %q: %v", raw, err))
	}
	return v
}
