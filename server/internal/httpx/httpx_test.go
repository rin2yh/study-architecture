package httpx

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

func TestNewEngine(t *testing.T) {
	if NewEngine() == nil {
		t.Fatal("NewEngine() = nil")
	}
}

func TestServe_GracefulShutdown(t *testing.T) {
	addr := freeAddr(t)
	engine := NewEngine()
	engine.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- Serve(ctx, addr, engine) }()

	// 固定 sleep は flaky なので、起動完了をポーリングで待つ。
	ready := false
	for range 100 {
		resp, err := http.Get("http://" + addr + "/ping")
		if err == nil {
			if cerr := resp.Body.Close(); cerr != nil {
				t.Logf("close ping body: %v", cerr)
			}
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("server did not become ready")
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Serve() = %v, want nil after graceful shutdown", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Serve did not return after context cancel")
	}
}

func TestServe_ListenError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 使用中のアドレスへ bind させると ListenAndServe が即座に error を返す。
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Logf("close listener: %v", err)
		}
	}()

	if err := Serve(ctx, l.Addr().String(), NewEngine()); err == nil {
		t.Fatal("Serve(): want error for in-use addr")
	}
}
