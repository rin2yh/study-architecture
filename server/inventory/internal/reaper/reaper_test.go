package reaper

import (
	"context"
	"errors"
	"testing"
)

type expirerStub struct {
	calls int
}

func (s *expirerStub) ExpireReservations(context.Context) error {
	s.calls++
	return nil
}

func TestRunStopsOnCanceledContext(t *testing.T) {
	s := &expirerStub{}
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
