// Package reaper は TTL 切れの予約を定期的に解放して在庫を回収する (ADR-[[202606262000]])。
package reaper

import (
	"context"
	"log/slog"
	"time"
)

type ExpiredReleaser interface {
	ReleaseExpiredReservations(ctx context.Context) error
}

type Reaper struct {
	releaser ExpiredReleaser
	interval time.Duration
}

func New(releaser ExpiredReleaser) *Reaper {
	return &Reaper{releaser: releaser, interval: time.Minute}
}

func (r *Reaper) Run(ctx context.Context) error {
	slog.Info("inventory reaper started", "interval", r.interval)
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		if err := r.releaser.ReleaseExpiredReservations(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// 集計は expires_at で既に在庫を除外済みなので、台帳追記が遅れても売り越しはしない。
			slog.Warn("inventory reaper: release expired failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}
