// Package strconvx は strconv の補助を共有層に置く。
package strconvx

import (
	"fmt"
	"strconv"
)

// 基数とビット幅を呼び出しごとに書かないための入口。
func ParseInt64(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func FormatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

// 呼び出し側が扱えない失敗を error にしないための契約。検証済みの値しか渡らない箇所に限って使う。
func MustInt64(raw string) int64 {
	v, err := ParseInt64(raw)
	if err != nil {
		panic(fmt.Sprintf("strconvx: invalid int64 %q: %v", raw, err))
	}
	return v
}
