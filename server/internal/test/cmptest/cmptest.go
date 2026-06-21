package cmptest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Equal[T any](t *testing.T, want, got T, ignoreFields ...string) {
	t.Helper()
	if d := cmp.Diff(want, got, opts[T](ignoreFields)...); d != "" {
		t.Fatalf("mismatch (-want +got):\n%s", d)
	}
}

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
