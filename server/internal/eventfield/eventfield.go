// Package eventfield は wire の payload から 1 フィールドを取り出す口を必須と任意の 2 つに絞る
// (ADR-[[202608160730]])。
package eventfield

import "fmt"

func Required[T any](values map[string]any, name string) (T, error) {
	var zero T
	v, ok := values[name].(T)
	if !ok {
		return zero, fmt.Errorf("%s: got %#v, want %T", name, values[name], zero)
	}
	return v, nil
}

func Optional[T any](values map[string]any, name string) (T, error) {
	var zero T
	raw, present := values[name]
	if !present {
		return zero, nil
	}
	v, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("%s: got %#v, want %T", name, raw, zero)
	}
	return v, nil
}
