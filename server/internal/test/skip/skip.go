// Package skip は結合テストの skip 条件をまとめる。
package skip

import "testing"

func Short(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skip integration test in -short mode")
	}
}
