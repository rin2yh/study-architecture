// Package cmptest は go-cmp による等値アサーションを 1 箇所に集約し、各テストの
// cmp.Diff 定型 (IgnoreFields / EquateEmpty / 失敗時の -want +got 出力) の重複を無くす。
package cmptest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Equal は構造体 1 件を比較する。ignoreFields には T のフィールド名 (生成 id や
// CreatedAt 等、検証対象外のもの) を渡す。
func Equal[T any](t *testing.T, want, got T, ignoreFields ...string) {
	t.Helper()
	if d := cmp.Diff(want, got, opts[T](ignoreFields)...); d != "" {
		t.Fatalf("mismatch (-want +got):\n%s", d)
	}
}

// EqualSlice はスライスを比較する。ignoreFields は要素型 E のフィールド名。
func EqualSlice[E any](t *testing.T, want, got []E, ignoreFields ...string) {
	t.Helper()
	if d := cmp.Diff(want, got, opts[E](ignoreFields)...); d != "" {
		t.Fatalf("mismatch (-want +got):\n%s", d)
	}
}

func opts[T any](ignoreFields []string) []cmp.Option {
	o := []cmp.Option{cmpopts.EquateEmpty()}
	if len(ignoreFields) > 0 {
		var zero T
		o = append(o, cmpopts.IgnoreFields(zero, ignoreFields...))
	}
	return o
}
