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

func (r *PaymentQuery) ListPayments(ctx context.Context) ([]db.PaymentPayment, error) {
	return r.q.ListPayments(ctx)
}

func (r *PaymentQuery) GetPayment(ctx context.Context, id int64) (db.PaymentPayment, error) {
	row, err := r.q.GetPayment(ctx, id)
	if err != nil {
		return db.PaymentPayment{}, dberr.FromRead(err)
	}
	return row, nil
}
