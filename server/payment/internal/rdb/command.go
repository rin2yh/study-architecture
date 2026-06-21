package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

type PaymentCommand struct {
	q db.Querier
}

func NewPaymentCommand(pool *pgxpool.Pool) *PaymentCommand {
	return &PaymentCommand{q: db.New(pool)}
}

func (r *PaymentCommand) CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.PaymentPayment, error) {
	return r.q.CreatePayment(ctx, arg)
}

func (r *PaymentCommand) UpdatePayment(ctx context.Context, arg db.UpdatePaymentParams) (db.PaymentPayment, error) {
	row, err := r.q.UpdatePayment(ctx, arg)
	if err != nil {
		return db.PaymentPayment{}, dberr.FromUpdate(err)
	}
	return row, nil
}
