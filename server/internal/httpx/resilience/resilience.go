// Package resilience は外向き HTTP 呼び出しに timeout / リトライ / サーキットブレーカを
// 被せる耐障害クライアントを提供する (ADR-[[202606261210]])。
package resilience

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

// Transport は base RoundTripper を timeout・リトライ・サーキットブレーカで包む。
// ブレーカはリトライの外側に置き、リトライを尽くした「論理 1 呼び出し」の成否を 1 件として数える
// (ADR-[[202606261210]])。
type Transport struct {
	base    http.RoundTripper
	breaker *gobreaker.CircuitBreaker[*http.Response]
	retry   RetryPolicy
	timeout time.Duration
	// 呼び出し先が冪等なら非冪等メソッド (POST) もリトライ対象に含める (ADR-[[202606261214]])。
	retryNonIdempotent bool
}

var _ http.RoundTripper = (*Transport)(nil)

// Option は Transport の任意設定。
type Option func(*Transport)

// RetryNonIdempotent は POST など非冪等メソッドのリトライを許可する。冪等性を担保した
// 呼び出し先 (例: idempotency key を持つ order→payment) だけに付ける (ADR-[[202606261214]])。
func RetryNonIdempotent() Option {
	return func(t *Transport) { t.retryNonIdempotent = true }
}

// otelhttp 計装を共用しないとサービス間呼び出しでトレースが切れる。リトライをその外側に置くことで、
// 各試行が独立した span として観測できる。
func NewClient(name string, opts ...Option) *http.Client {
	base := otelhttp.NewTransport(http.DefaultTransport)
	return &http.Client{Transport: newTransport(name, base, DefaultResilienceConfig(), opts...)}
}

func newTransport(name string, base http.RoundTripper, cfg ResilienceConfig, opts ...Option) *Transport {
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
	t := &Transport{base: base, breaker: breaker, retry: cfg.Retry, timeout: cfg.AttemptTimeout}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.breaker.Execute(func() (*http.Response, error) {
		return t.roundTripWithRetry(req)
	})
}

func (t *Transport) roundTripWithRetry(req *http.Request) (*http.Response, error) {
	attempts := 1
	// 素朴な POST リトライは二重決済を生むため、安全なメソッドか、冪等性を担保した呼び出し先
	// (RetryNonIdempotent) かつ body を巻き戻せる場合に限る (ADR-[[202606261210]], ADR-[[202606261214]])。
	if safeToRetry(req.Method) || (t.retryNonIdempotent && rewindable(req)) {
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
			if err := rewindBody(req); err != nil {
				return nil, err
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

func (t *Transport) attempt(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), t.timeout)
	resp, err := t.base.RoundTrip(req.WithContext(ctx))
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

func (t *Transport) backoff(retry int) time.Duration {
	d := t.retry.BaseDelay << (retry - 1)
	if d > t.retry.MaxDelay {
		d = t.retry.MaxDelay
	}
	// 一斉リトライによる thundering herd を避けるため。
	return time.Duration(rand.Float64() * float64(d))
}

// GetBody が無い body は再送で復元できず、リトライすると空ボディを送ってしまう。
func rewindable(req *http.Request) bool {
	return req.Body == nil || req.GetBody != nil
}

func rewindBody(req *http.Request) error {
	if req.Body == nil || req.GetBody == nil {
		return nil
	}
	body, err := req.GetBody()
	if err != nil {
		return err
	}
	req.Body = body
	return nil
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
