// Package event は payment のドメインイベントの語彙を定める。
package event

// IsSettled は status が「決済確定」を表すかを判定する。確定の語彙はサービス間で揺れるため
// (capture/settle/paid 相当)、配送手配のトリガとなる確定状態をここで一元的に定義する。
func IsSettled(status string) bool {
	switch status {
	case "paid", "settled", "captured":
		return true
	default:
		return false
	}
}
