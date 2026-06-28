package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/orderevent"
	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type OutboxStore struct {
	q db.Querier
}

var _ outbox.Store = (*OutboxStore)(nil)

func NewOutboxStore(pool *pgxpool.Pool) *OutboxStore {
	return &OutboxStore{q: db.New(pool)}
}

func (s *OutboxStore) FetchUnpublished(ctx context.Context, limit int) ([]outbox.Message, error) {
	rows, err := s.q.ListUnpublishedCancelledEvents(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	msgs := make([]outbox.Message, 0, len(rows))
	for _, r := range rows {
		values := orderevent.Cancelled{OrderID: r.ID}.Values()
		// 発行時に保持した traceparent を送出メッセージへ戻し、consumer 側の span link を切らさない。
		if r.CancelledEventTraceparent != "" {
			values[orderevent.FieldTraceparent] = r.CancelledEventTraceparent
		}
		msgs = append(msgs, outbox.Message{ID: r.ID, Stream: orderevent.Stream, Values: values})
	}
	return msgs, nil
}

func (s *OutboxStore) MarkPublished(ctx context.Context, id int64) error {
	return s.q.MarkCancelledEventPublished(ctx, id)
}
