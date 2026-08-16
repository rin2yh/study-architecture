// Package eventfield は wire の payload から 1 フィールドを取り出す口を、必須と任意の 2 つだけに
// 絞る (ADR-[[202608160730]])。廃止の 1 段階目を Required から Optional への 1 語の変更で済ませ、
// 「欠落は許すが型違いは弾く」という間違えやすい条件を各 wire 契約に手書きさせない。
package eventfield

import "fmt"

// Required は欠落・型違いのどちらも error にする。ゼロ値へ化けたまま処理が進むのを防ぐ。
func Required[T any](values map[string]any, name string) (T, error) {
	var zero T
	v, ok := values[name].(T)
	if !ok {
		return zero, fmt.Errorf("%s: got %#v, want %T", name, values[name], zero)
	}
	return v, nil
}

// Optional は欠落だけを許す。旧 producer がまだ発行していない追加直後と、これから消す廃止予定が
// この状態にあたる。型違いは契約違反なので Required と同じく弾く。
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
