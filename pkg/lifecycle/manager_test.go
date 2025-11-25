package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestManager_AddWorker(t *testing.T) {
	m := NewManager()

	err := m.AddWorker("test", func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Fatalf("添加协程失败: %v", err)
	}

	err = m.AddWorker("test", func(ctx context.Context) error {
		return nil
	})

	if err != ErrWorkerExists {
		t.Errorf("期望 ErrWorkerExists, got %v", err)
	}
}

func TestManager_RemoveWorker(t *testing.T) {
	m := NewManager()

	m.AddWorker("test", func(ctx context.Context) error {
		return nil
	})

	err := m.RemoveWorker("test")
	if err != nil {
		t.Fatalf("移除协程失败: %v", err)
	}

	err = m.RemoveWorker("test")
	if err != ErrWorkerNotFound {
		t.Errorf("期望 ErrWorkerNotFound, got %v", err)
	}
}

func TestManager_Hooks(t *testing.T) {
	m := NewManager(WithShutdownTimeout(1 * time.Second))

	startupCalled := false
	workerStartCalled := false
	workerExitCalled := false
	shutdownCalled := false

	m.OnStartup(func(ctx context.Context) error {
		startupCalled = true
		return nil
	})

	m.OnWorkerStart(func(name string, err error) {
		workerStartCalled = true
	})

	m.OnWorkerExit(func(name string, err error) {
		workerExitCalled = true
	})

	m.OnShutdown(func(ctx context.Context) error {
		shutdownCalled = true
		return nil
	})

	m.AddWorker("test", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	go func() {
		time.Sleep(100 * time.Millisecond)
		m.Shutdown()
	}()

	m.Run()

	if !startupCalled {
		t.Error("OnStartup 未被调用")
	}
	if !workerStartCalled {
		t.Error("OnWorkerStart 未被调用")
	}
	if !workerExitCalled {
		t.Error("OnWorkerExit 未被调用")
	}
	if !shutdownCalled {
		t.Error("OnShutdown 未被调用")
	}
}

func TestManager_WorkerError(t *testing.T) {
	m := NewManager(WithShutdownTimeout(1 * time.Second))

	expectedErr := errors.New("worker error")

	m.AddWorker("test", func(ctx context.Context) error {
		return expectedErr
	})

	err := m.Run()
	if err != expectedErr {
		t.Errorf("期望错误 %v, got %v", expectedErr, err)
	}
}

func TestManager_ContextCancellation(t *testing.T) {
	m := NewManager(WithShutdownTimeout(1 * time.Second))

	cancelled := false

	m.AddWorker("test", func(ctx context.Context) error {
		<-ctx.Done()
		cancelled = true
		return nil
	})

	go func() {
		time.Sleep(100 * time.Millisecond)
		m.Shutdown()
	}()

	m.Run()

	if !cancelled {
		t.Error("协程未收到取消信号")
	}
}

func TestWorker_StopFunc(t *testing.T) {
	stopCalled := false

	worker := NewWorker("test",
		func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
		WithStopFunc(func(ctx context.Context) error {
			stopCalled = true
			return nil
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	go worker.Run(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	worker.Stop(context.Background())

	if !stopCalled {
		t.Error("StopFunc 未被调用")
	}
}
