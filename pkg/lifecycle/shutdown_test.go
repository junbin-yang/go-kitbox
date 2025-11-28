package lifecycle

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestManager_HTTPServerShutdown(t *testing.T) {
	m := NewManager(WithShutdownTimeout(5 * time.Second))

	server := &http.Server{Addr: ":18080"}

	_ = m.AddWorker("http-server",
		func(ctx context.Context) error {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}
			return nil
		},
		WithStopFunc(func(ctx context.Context) error {
			return server.Shutdown(ctx)
		}),
	)

	shutdownCalled := false
	m.OnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	done := make(chan error, 1)
	go func() {
		done <- m.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	if err := m.Shutdown(); err != nil {
		t.Errorf("Shutdown 返回错误: %v", err)
	}
	elapsed := time.Since(start)

	<-done

	if elapsed > 2*time.Second {
		t.Errorf("Shutdown 耗时过长: %v", elapsed)
	}

	if !shutdownCalled {
		t.Error("OnShutdown 未被调用")
	}
}
