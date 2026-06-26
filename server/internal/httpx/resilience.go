package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// 4xx は確定応答なので errRetryable で包まず、ブレーカを開かせない。
var errRetryable = errors.New("retryable upstream failure")

// RetryPolicy は冪等な呼び出しの指数バックオフ + フルジッタの設定。
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// ResilienceConfig は 1 呼び出し先ぶんの timeout / リトライ / ブレーカ設定。
type ResilienceConfig struct {
	Retry            RetryPolicy
	AttemptTimeout   time.Duration // 1 試行あたりの上限
	BreakerTimeout   time.Duration // オープン状態を保つ期間
	BreakerThreshold uint32        // この連続失敗数でオープンへ倒す
}

func DefaultResilienceConfig() ResilienceConfig {
	return ResilienceConfig{
		Retry:            RetryPolicy{MaxAttempts: 3, BaseDelay: 50 * time.Millisecond, MaxDelay: time.Second},
		AttemptTimeout:   3 * time.Second,
		BreakerTimeout:   5 * time.Second,
		BreakerThreshold: 5,
	}
}

// ResilientTransport は base RoundTripper を timeout・リトライ・サーキットブレーカで包む。
// ブレーカはリトライの外側に置き、リトライを尽くした「論理 1 呼び出し」の成否を 1 件として数える
// (ADR-[[202606261210]])。
type ResilientTransport struct {
	base    http.RoundTripper
	breaker *gobreaker.CircuitBreaker[*http.Response]
	retry   RetryPolicy
	timeout time.Duration
}

var _ http.RoundTripper = (*ResilientTransport)(nil)

// 各サービス間呼び出しは otelhttp 計装したクライアントを共用しないとトレースが切れるため、base に
// otelhttp transport を据える。リトライをその外側に置くことで、各試行が独立した span として観測できる。
func NewResilientClient(name string) *http.Client {
	return newResilientClient(name, otelhttp.NewTransport(http.DefaultTransport), DefaultResilienceConfig())
}

func newResilientClient(name string, base http.RoundTripper, cfg ResilienceConfig) *http.Client {
	return &http.Client{Transport: newResilientTransport(name, base, cfg)}
}

func newResilientTransport(name string, base http.RoundTripper, cfg ResilienceConfig) *ResilientTransport {
	breaker := gobreaker.NewCircuitBreaker[*http.Response](gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Timeout:     cfg.BreakerTimeout,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= cfg.BreakerThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state changed", "breaker", name, "from", from.String(), "to", to.String())
		},
	})
	return &ResilientTransport{base: base, breaker: breaker, retry: cfg.Retry, timeout: cfg.AttemptTimeout}
}

func (t *ResilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.breaker.Execute(func() (*http.Response, error) {
		return t.roundTripWithRetry(req)
	})
}

func (t *ResilientTransport) roundTripWithRetry(req *http.Request) (*http.Response, error) {
	attempts := 1
	// 非冪等な POST を素朴にリトライすると二重決済を生むため、冪等性 (ADR-[[202606261214]]) が
	// 入るまではリトライを安全なメソッドに限る (ADR-[[202606261210]])。
	if safeToRetry(req.Method) {
		attempts = t.retry.MaxAttempts
	}

	var lastErr error
	for i := range attempts {
		if i > 0 {
			select {
			case <-time.After(t.backoff(i)):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
		resp, err := t.attempt(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// 呼び出し元が諦めたなら追加試行しない。
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}
	}
	return nil, lastErr
}

func (t *ResilientTransport) attempt(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), t.timeout)
	resp, err := t.base.RoundTrip(req.Clone(ctx))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("%w: %v", errRetryable, err)
	}
	if retryableStatus(resp.StatusCode) {
		drainAndClose(resp.Body)
		cancel()
		return nil, fmt.Errorf("%w: upstream status %d", errRetryable, resp.StatusCode)
	}
	// 2xx / 4xx は確定応答。body 読み出し中に試行 timeout で context を切らさないよう、Close 時に cancel する。
	resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

func (t *ResilientTransport) backoff(retry int) time.Duration {
	d := float64(t.retry.BaseDelay) * math.Pow(2, float64(retry-1))
	d = math.Min(d, float64(t.retry.MaxDelay))
	// フルジッタ [0, d): 一斉リトライによる thundering herd を散らす。
	return time.Duration(rand.Float64() * d)
}

func safeToRetry(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func retryableStatus(code int) bool {
	switch code {
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

// cancelOnClose は body を読み終えて Close した時点で試行の timeout context を解放する。
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	c.cancel()
	return c.ReadCloser.Close()
}

// drainAndClose は接続を再利用させるため body を読み切って閉じる。再利用は best-effort で、
// 失敗しても新規接続を張るだけなのでエラーは捨てる。
func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 4096))
	_ = body.Close()
}
