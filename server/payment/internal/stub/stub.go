package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

type Repo struct {
	Payments []db.PaymentPayment
	Err      error
}

func (s Repo) ListPayments(context.Context) ([]db.PaymentPayment, error) {
	return s.Payments, s.Err
}
