package resilience

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
)

func scriptedServer(t *testing.T, statuses ...int) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := hits.Add(1)
		i := int(n) - 1
		if i >= len(statuses) {
			i = len(statuses) - 1
		}
		w.WriteHeader(statuses[i])
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func fastConfig() ResilienceConfig {
	return ResilienceConfig{
		Retry:            RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 4 * time.Millisecond},
		AttemptTimeout:   500 * time.Millisecond,
		BreakerTimeout:   20 * time.Millisecond,
		BreakerThreshold: 2,
	}
}

func doRoundTrip(t *testing.T, tr *ResilientTransport, method, url string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := tr.RoundTrip(req)
	if resp != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	return resp, err
}

func TestResilientTransportRetry(t *testing.T) {
	type want struct {
		status  int
		hits    int64
		wantErr bool
	}
	tests := []struct {
		name     string
		method   string
		statuses []int
		want     want
	}{
		{"正常系 GET 初回成功は 1 回で 200", http.MethodGet, []int{http.StatusOK}, want{status: 200, hits: 1}},
		{"正常系 GET 2 回失敗後に成功で 3 回叩く", http.MethodGet, []int{http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusOK}, want{status: 200, hits: 3}},
		{"準正常系 GET 4xx は確定応答でリトライしない", http.MethodGet, []int{http.StatusNotFound}, want{status: 404, hits: 1}},
		{"準正常系 POST は非冪等でリトライしない", http.MethodPost, []int{http.StatusServiceUnavailable}, want{hits: 1, wantErr: true}},
		{"異常系 GET 5xx 継続は MaxAttempts まで叩いて失敗", http.MethodGet, []int{http.StatusInternalServerError}, want{hits: 3, wantErr: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, hits := scriptedServer(t, tt.statuses...)
			tr := newResilientTransport(tt.name, http.DefaultTransport, fastConfig())

			resp, err := doRoundTrip(t, tr, tt.method, srv.URL)
			if tt.want.wantErr {
				if err == nil {
					t.Fatalf("err = nil, want error")
				}
				if !errors.Is(err, errRetryable) {
					t.Fatalf("err = %v, want errRetryable", err)
				}
			} else {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				if resp.StatusCode != tt.want.status {
					t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want.status)
				}
			}
			if got := hits.Load(); got != tt.want.hits {
				t.Fatalf("hits = %d, want %d", got, tt.want.hits)
			}
		})
	}
}

func TestResilientTransportRetryNonIdempotent(t *testing.T) {
	const payload = `{"orderId":1}`
	type args struct {
		opts []Option
		body io.Reader
	}
	type want struct {
		hits int64
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 opt-in POST は body を巻き戻して MaxAttempts まで再送", args{[]Option{RetryNonIdempotent()}, strings.NewReader(payload)}, want{hits: 3}},
		// GetBody が無い body は再送で空になるため、opt-in でもリトライさせない。
		{"準正常系 opt-in でも GetBody 無し body はリトライしない", args{[]Option{RetryNonIdempotent()}, struct{ io.Reader }{strings.NewReader(payload)}}, want{hits: 1}},
		{"準正常系 opt-in 無し POST はリトライしない", args{nil, strings.NewReader(payload)}, want{hits: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var bodies []string
			var hits atomic.Int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				mu.Lock()
				bodies = append(bodies, string(b))
				mu.Unlock()
				hits.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(srv.Close)

			tr := newResilientTransport(tt.name, http.DefaultTransport, fastConfig(), tt.args.opts...)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, tt.args.body)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			resp, err := tr.RoundTrip(req)
			if resp != nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
			}
			if err == nil {
				t.Fatal("err = nil, want error (all attempts 503)")
			}
			if got := hits.Load(); got != tt.want.hits {
				t.Fatalf("hits = %d, want %d", got, tt.want.hits)
			}
			mu.Lock()
			defer mu.Unlock()
			for i, b := range bodies {
				if b != payload {
					t.Fatalf("attempt %d body = %q, want %q (rewind failed)", i, b, payload)
				}
			}
		})
	}
}

func TestResilientTransportCircuitBreaker(t *testing.T) {
	cfg := fastConfig()
	cfg.Retry.MaxAttempts = 1 // 論理失敗とサーバ被弾を 1:1 にして閾値を読みやすくする
	srv, hits := scriptedServer(t, http.StatusServiceUnavailable)
	tr := newResilientTransport("cb", http.DefaultTransport, cfg)

	for range int(cfg.BreakerThreshold) {
		if _, err := doRoundTrip(t, tr, http.MethodGet, srv.URL); err == nil {
			t.Fatal("err = nil, want failure while tripping breaker")
		}
	}
	hitsAtOpen := hits.Load()

	_, err := doRoundTrip(t, tr, http.MethodGet, srv.URL)
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Fatalf("err = %v, want ErrOpenState", err)
	}
	if got := hits.Load(); got != hitsAtOpen {
		t.Fatalf("hits = %d while open, want unchanged %d", got, hitsAtOpen)
	}

	srv2, _ := scriptedServer(t, http.StatusOK)
	time.Sleep(cfg.BreakerTimeout + 5*time.Millisecond)
	if _, err := doRoundTrip(t, tr, http.MethodGet, srv2.URL); err != nil {
		t.Fatalf("err = %v, want recovery after breaker timeout", err)
	}
	if tr.breaker.State() != gobreaker.StateClosed {
		t.Fatalf("state = %v, want closed after success", tr.breaker.State())
	}
}

func TestResilientTransportConcurrent(t *testing.T) {
	srv, _ := scriptedServer(t, http.StatusOK)
	tr := newResilientTransport("concurrent", http.DefaultTransport, fastConfig())

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := doRoundTrip(t, tr, http.MethodGet, srv.URL); err != nil {
				t.Errorf("concurrent RoundTrip: %v", err)
			}
		}()
	}
	wg.Wait()
}
