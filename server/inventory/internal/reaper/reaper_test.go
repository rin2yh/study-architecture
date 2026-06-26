package reaper

import (
	"context"
	"errors"
	"testing"
)

type releaserStub struct {
	calls int
}

func (s *releaserStub) ReleaseExpiredReservations(context.Context) error {
	s.calls++
	return nil
}

func TestRunStopsOnCanceledContext(t *testing.T) {
	s := &releaserStub{}
	r := New(s)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := r.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
	if s.calls == 0 {
		t.Fatal("releaser must drain once before waiting for the first tick")
	}
}
