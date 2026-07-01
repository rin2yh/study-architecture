package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/payment/internal/db"
	"github.com/rin2yh/study-architecture/server/payment/internal/rdb"
)

type PaymentStub struct {
	Payments []db.PaymentPayment
	Payment  db.PaymentPayment
	Err      error
}

func (s PaymentStub) ListPayments(context.Context) ([]db.PaymentPayment, error) {
	return s.Payments, s.Err
}

func (s PaymentStub) GetPayment(context.Context, int64) (db.PaymentPayment, error) {
	return s.Payment, s.Err
}

func (s PaymentStub) CreatePayment(context.Context, db.CreatePaymentParams) (db.PaymentPayment, error) {
	return s.Payment, s.Err
}

func (s PaymentStub) UpdatePayment(context.Context, rdb.PaymentUpdate) (db.PaymentPayment, error) {
	return s.Payment, s.Err
}
