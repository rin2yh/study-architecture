// Package assert はテスト用の共通アサーションヘルパー。
package assert

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func DeepEqual[T any](t *testing.T, want, got T, ignoreFields ...string) {
	t.Helper()
	if d := cmp.Diff(want, got, opts[T](ignoreFields)...); d != "" {
		t.Fatalf("mismatch (-want +got):\n%s", d)
	}
}

func DeepEqualSlice[E any](t *testing.T, want, got []E, ignoreFields ...string) {
	t.Helper()
	if d := cmp.Diff(want, got, opts[E](ignoreFields)...); d != "" {
		t.Fatalf("mismatch (-want +got):\n%s", d)
	}
}

func ErrorCode(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if e.Code != wantCode {
		t.Fatalf("code = %q, want %q", e.Code, wantCode)
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
