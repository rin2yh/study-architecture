// Package skip は結合テストの skip 条件をまとめる。
package skip

import "testing"

// Short は -short 実行時に結合テストを skip する。
func Short(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skip integration test in -short mode")
	}
}
