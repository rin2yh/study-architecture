package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/payment/internal/rdb"
)

type PaymentStub struct {
	Payments []rdb.Payment
	Payment  rdb.Payment
	Err      error
}

func (s PaymentStub) ListPayments(context.Context) ([]rdb.Payment, error) {
	return s.Payments, s.Err
}

func (s PaymentStub) GetPayment(context.Context, int64) (rdb.Payment, error) {
	return s.Payment, s.Err
}

func (s PaymentStub) CreatePayment(context.Context, rdb.PaymentCreate) (rdb.Payment, error) {
	return s.Payment, s.Err
}

func (s PaymentStub) UpdatePayment(context.Context, rdb.PaymentUpdate) (rdb.Payment, error) {
	return s.Payment, s.Err
}
