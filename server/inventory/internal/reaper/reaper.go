// Package reaper は TTL 切れの予約を定期的に解放して在庫を回収する (ADR-[[202606262000]])。
package reaper

import (
	"context"
	"log/slog"
	"time"
)

type ReservationExpirer interface {
	ExpireReservations(ctx context.Context) error
}

type Reaper struct {
	expirer  ReservationExpirer
	interval time.Duration
}

func New(expirer ReservationExpirer) *Reaper {
	return &Reaper{expirer: expirer, interval: time.Minute}
}

func (r *Reaper) Run(ctx context.Context) error {
	slog.Info("inventory reaper started", "interval", r.interval)
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		if err := r.expirer.ExpireReservations(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// 集計は expires_at で既に在庫を除外済みなので、台帳追記が遅れても売り越しはしない。
			slog.Warn("inventory reaper: expire reservations failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}
