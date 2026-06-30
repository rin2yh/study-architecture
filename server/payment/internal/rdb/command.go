package rdb

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

type PaymentCommand struct {
	pool *pgxpool.Pool
	q    db.Querier
}

func NewPaymentCommand(pool *pgxpool.Pool) *PaymentCommand {
	return &PaymentCommand{pool: pool, q: db.New(pool)}
}

func (r *PaymentCommand) CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.PaymentPayment, error) {
	row, err := r.q.CreatePayment(ctx, arg)
	if errors.Is(dberr.FromInsertSkipped(err), dberr.ErrConflict) {
		// 同一キーの再送 = 既に作成済み。冪等に既存決済を成功として返す (ADR-[[202606261214]])。
		return r.q.GetPaymentByIdempotencyKey(ctx, arg.IdempotencyKey)
	}
	return row, err
}

// ADR-[[202606300600]]
func (r *PaymentCommand) UpdatePayment(ctx context.Context, id int64, status string, settle bool, traceparent string) (db.PaymentPayment, error) {
	// 確定でなければ outbox 投入がなく調整する 2 書き込みがないので、単文のままにする。
	if !settle {
		row, err := r.q.UpdatePayment(ctx, db.UpdatePaymentParams{ID: id, Status: status})
		if err != nil {
			return db.PaymentPayment{}, dberr.FromUpdate(err)
		}
		return row, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.PaymentPayment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	row, err := qtx.UpdatePayment(ctx, db.UpdatePaymentParams{ID: id, Status: status})
	if err != nil {
		return db.PaymentPayment{}, dberr.FromUpdate(err)
	}
	payload, err := json.Marshal(paymentevent.Settled{
		PaymentID:   row.ID,
		OrderID:     row.OrderID,
		AmountCents: row.AmountCents,
	}.Values())
	if err != nil {
		return db.PaymentPayment{}, err
	}
	if err := qtx.InsertOutbox(ctx, db.InsertOutboxParams{
		AggregateID: row.ID,
		EventType:   paymentevent.TypeSettled,
		Payload:     payload,
		Traceparent: traceparent,
	}); err != nil {
		return db.PaymentPayment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.PaymentPayment{}, err
	}
	return row, nil
}

// RefundByOrder は order.cancelled の補償。確定済みは返金・未確定はキャンセルへ倒す。状態ガード付き
// UPDATE で冪等にし、再配信での二重返金を防ぐ (ADR-[[202606261702]] / ADR-[[202606261214]])。
func (r *PaymentCommand) RefundByOrder(ctx context.Context, orderID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	if err := qtx.RefundPaymentByOrder(ctx, orderID); err != nil {
		return err
	}
	if err := qtx.VoidPendingPaymentByOrder(ctx, orderID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
