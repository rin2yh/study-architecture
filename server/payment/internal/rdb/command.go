package rdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/order"
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

func (r *PaymentCommand) CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.PaymentPayment, error) {
	row, err := r.q.CreatePayment(ctx, arg)
	if errors.Is(dberr.FromInsertSkipped(err), dberr.ErrConflict) {
		// 同一キーの再送 = 既に作成済み。冪等に既存決済を成功として返す (ADR-[[202606261214]])。
		return r.q.GetPaymentByIdempotencyKey(ctx, arg.IdempotencyKey)
	}
	return row, err
}

// PaymentUpdate は status 更新と、確定時に発行する settled イベントの指定をまとめる。
type PaymentUpdate struct {
	ID          int64
	Status      string
	Settle      bool
	Traceparent string
}

// ADR-[[202606300600]]
func (r *PaymentCommand) UpdatePayment(ctx context.Context, u PaymentUpdate) (db.PaymentPayment, error) {
	// 確定でなければ outbox 投入がなく調整する 2 書き込みがないので、単文のままにする。
	if !u.Settle {
		row, err := r.q.UpdatePayment(ctx, db.UpdatePaymentParams{ID: u.ID, Status: u.Status})
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
	row, err := qtx.UpdatePayment(ctx, db.UpdatePaymentParams{ID: u.ID, Status: u.Status})
	if err != nil {
		return db.PaymentPayment{}, dberr.FromUpdate(err)
	}
	orderID, err := order.New(row.OrderID)
	if err != nil {
		return db.PaymentPayment{}, err
	}
	ev := paymentevent.Settled{PaymentID: row.ID, OrderID: orderID, AmountCents: row.AmountCents}
	if err := outbox.Dispatch(ctx, outboxInserter{qtx}, u.Traceparent, ev); err != nil {
		return db.PaymentPayment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.PaymentPayment{}, err
	}
	return row, nil
}

// RefundByOrder は order.cancelled の補償。確定済みは返金・未確定はキャンセルへ倒す。状態ガード付き
// UPDATE で冪等にし、再配信での二重返金を防ぐ (ADR-[[202606261702]] / ADR-[[202606261214]])。
func (r *PaymentCommand) RefundByOrder(ctx context.Context, orderID order.ID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	if err := qtx.RefundPaymentByOrder(ctx, orderID.Int64()); err != nil {
		return err
	}
	if err := qtx.VoidPendingPaymentByOrder(ctx, orderID.Int64()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
