// Package paymentevent は payment→shipping の決済確定イベントの wire 契約を一元的に定める。
// producer (payment) と consumer (shipping) が文字列を各自で持つと無言で配送経路が切れるため、
// stream 名・イベント種別・フィールドキー・ペイロードをここだけに置く。
package paymentevent

const (
	Stream      = "payment.events"
	TypeSettled = "payment.settled"
)

const (
	FieldEvent       = "event"
	FieldPaymentID   = "paymentId"
	FieldOrderID     = "orderId"
	FieldAmountCents = "amountCents"
)

type Settled struct {
	PaymentID   int64
	OrderID     int64
	AmountCents int64
}

func (s Settled) Values() map[string]any {
	return map[string]any{
		FieldEvent:       TypeSettled,
		FieldPaymentID:   s.PaymentID,
		FieldOrderID:     s.OrderID,
		FieldAmountCents: s.AmountCents,
	}
}
