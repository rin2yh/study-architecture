package rdb

import (
	"time"

	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

// port を sqlc 生成型から切り離す (ADR-[[202607011200]])。idempotency_key など API に出さない
// 内部列は載せず、pgtype も標準型へ寄せる。
type Payment struct {
	ID          int64
	OrderID     int64
	AmountCents int64
	Method      string
	Status      string
	CreatedAt   time.Time
}

type PaymentCreate struct {
	OrderID        int64
	AmountCents    int64
	Method         string
	Status         string
	IdempotencyKey string
}

func toPayment(r db.PaymentPayment) Payment {
	return Payment{
		ID:          r.ID,
		OrderID:     r.OrderID,
		AmountCents: r.AmountCents,
		Method:      r.Method,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt.Time,
	}
}

func toPayments(rows []db.PaymentPayment) []Payment {
	out := make([]Payment, 0, len(rows))
	for _, r := range rows {
		out = append(out, toPayment(r))
	}
	return out
}
