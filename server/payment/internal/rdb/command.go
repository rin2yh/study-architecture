package rdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

// sqlc 生成型を共有層に持ち込まないためのアダプタ。
type outboxInserter struct{ q db.Querier }

func (o outboxInserter) InsertOutbox(ctx context.Context, row outbox.Row) error {
	return o.q.InsertOutbox(ctx, db.InsertOutboxParams{
		AggregateID: row.AggregateID,
		EventType:   row.EventType,
		Payload:     row.Payload,
		Traceparent: row.Traceparent,
	})
}

type PaymentCommand struct {
	pool *pgxpool.Pool
	q    db.Querier
}

func NewPaymentCommand(pool *pgxpool.Pool) *PaymentCommand {
	return &PaymentCommand{pool: pool, q: db.New(pool)}
}

func (r *PaymentCommand) CreatePayment(ctx context.Context, arg PaymentCreate) (Payment, error) {
	row, err := r.q.CreatePayment(ctx, db.CreatePaymentParams{
		OrderID:        arg.OrderID,
		AmountCents:    arg.AmountCents,
		Method:         arg.Method,
		Status:         arg.Status,
		IdempotencyKey: arg.IdempotencyKey,
	})
	if errors.Is(dberr.FromInsertSkipped(err), dberr.ErrConflict) {
		// ADR-[[202606261214]]
		existing, err := r.q.GetPaymentByIdempotencyKey(ctx, arg.IdempotencyKey)
		return toPayment(existing), err
	}
	if err != nil {
		return Payment{}, err
	}
	return toPayment(row), nil
}

// PaymentUpdate は status 更新と、確定時に発行する settled イベントの指定をまとめる。
type PaymentUpdate struct {
	ID          int64
	Status      string
	Settle      bool
	Traceparent string
}

// ADR-[[202606300600]]
func (r *PaymentCommand) UpdatePayment(ctx context.Context, u PaymentUpdate) (Payment, error) {
	// 確定でなければ outbox 投入がなく調整する 2 書き込みがないので、単文のままにする。
	if !u.Settle {
		row, err := r.q.UpdatePayment(ctx, db.UpdatePaymentParams{ID: u.ID, Status: u.Status})
		if err != nil {
			return Payment{}, dberr.FromUpdate(err)
		}
		return toPayment(row), nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	row, err := qtx.UpdatePayment(ctx, db.UpdatePaymentParams{ID: u.ID, Status: u.Status})
	if err != nil {
		return Payment{}, dberr.FromUpdate(err)
	}
	ev := paymentevent.Settled{PaymentID: row.ID, OrderID: row.OrderID, AmountCents: row.AmountCents}
	if err := outbox.Dispatch(ctx, outboxInserter{qtx}, u.Traceparent, ev); err != nil {
		return Payment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}
	return toPayment(row), nil
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
