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

	m.AddWorker("http-server",
		func(ctx context.Context) error {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}
			return nil
		},
		WithStopFunc(func(ctx context.Context) error {
			t.Log("调用 server.Shutdown")
			return server.Shutdown(ctx)
		}),
	)

	shutdownCalled := false
	m.OnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		t.Log("OnShutdown 被调用")
		return nil
	})

	// 启动服务器
	go func() {
		if err := m.Run(); err != nil {
			t.Logf("Run 返回错误: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 触发退出
	t.Log("触发 Shutdown")
	start := time.Now()
	if err := m.Shutdown(); err != nil {
		t.Errorf("Shutdown 返回错误: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Shutdown 耗时: %v", elapsed)

	if elapsed > 2*time.Second {
		t.Errorf("Shutdown 耗时过长: %v, 应该立即退出", elapsed)
	}

	if !shutdownCalled {
		t.Error("OnShutdown 未被调用")
	}
}
