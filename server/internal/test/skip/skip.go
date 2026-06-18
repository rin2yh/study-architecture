// Package skip は結合テストの skip 条件をまとめる。
package skip

import "testing"

// Short は -short 実行時に結合テストを skip する。DB を持たない per-service の単体ジョブは
// -short で回るため、各結合テストの先頭で呼んで skip させる。
func Short(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skip integration test in -short mode")
	}
}
