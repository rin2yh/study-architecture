package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

type Repo struct {
	Payments []db.PaymentPayment
	Payment  db.PaymentPayment
	Err      error
}

func (s Repo) ListPayments(context.Context) ([]db.PaymentPayment, error) {
	return s.Payments, s.Err
}

func (s Repo) GetPayment(context.Context, int64) (db.PaymentPayment, error) {
	return s.Payment, s.Err
}

func (s Repo) CreatePayment(context.Context, db.CreatePaymentParams) (db.PaymentPayment, error) {
	return s.Payment, s.Err
}
