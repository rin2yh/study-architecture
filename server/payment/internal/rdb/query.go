package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

type PaymentQuery struct {
	q db.Querier
}

func NewPaymentQuery(pool *pgxpool.Pool) *PaymentQuery {
	return &PaymentQuery{q: db.New(pool)}
}

func (r *PaymentQuery) ListPayments(ctx context.Context) ([]Payment, error) {
	rows, err := r.q.ListPayments(ctx)
	if err != nil {
		return nil, err
	}
	return toPayments(rows), nil
}

func (r *PaymentQuery) GetPayment(ctx context.Context, id int64) (Payment, error) {
	row, err := r.q.GetPayment(ctx, id)
	if err != nil {
		return Payment{}, dberr.FromRead(err)
	}
	return toPayment(row), nil
}
