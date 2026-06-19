package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

type PaymentQuery struct {
	q db.Querier
}

type PaymentCommand struct {
	q db.Querier
}

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

func NewPaymentQuery(pool *pgxpool.Pool) *PaymentQuery {
	return &PaymentQuery{q: db.New(pool)}
}

func NewPaymentCommand(pool *pgxpool.Pool) *PaymentCommand {
	return &PaymentCommand{q: db.New(pool)}
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
