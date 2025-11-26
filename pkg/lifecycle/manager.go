package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Manager 生命周期管理器
type Manager struct {
	mu              sync.RWMutex
	workers         map[string]*Worker
	workerContexts  map[string]context.CancelFunc
	workerOrder     []string
	hooks           *Hooks
	signals         []os.Signal
	shutdownTimeout time.Duration
	rootCtx         context.Context
	cancel          context.CancelFunc
	running         bool
	wg              sync.WaitGroup
	errChan         chan error
}

// NewManager 创建生命周期管理器
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		workers:         make(map[string]*Worker),
		workerContexts:  make(map[string]context.CancelFunc),
		workerOrder:     make([]string, 0),
		hooks:           newHooks(),
		signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		shutdownTimeout: 30 * time.Second,
		rootCtx:         context.Background(),
		errChan:         make(chan error, 1),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// AddWorker 添加协程（运行时动态添加）
func (m *Manager) AddWorker(name string, runFunc RunFunc, opts ...WorkerOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.workers[name]; exists {
		return ErrWorkerExists
	}

	worker := NewWorker(name, runFunc, opts...)
	m.workers[name] = worker
	m.workerOrder = append(m.workerOrder, name)

	// 如果管理器已运行，立即启动该协程
	if m.running {
		workerCtx, workerCancel := context.WithCancel(m.rootCtx)
		m.workerContexts[name] = workerCancel
		m.wg.Add(1)

		go func(w *Worker, wCtx context.Context, wName string) {
			defer func() {
				m.wg.Done()
				m.mu.Lock()
				delete(m.workerContexts, wName)
				delete(m.workers, wName)
				for i, n := range m.workerOrder {
					if n == wName {
						m.workerOrder = append(m.workerOrder[:i], m.workerOrder[i+1:]...)
						break
					}
				}
				m.mu.Unlock()
			}()

			m.hooks.callWorkerStart(w.Name(), nil)
			err := w.Run(wCtx)
			m.hooks.callWorkerExit(w.Name(), err)

			if err != nil && err != context.Canceled {
				select {
				case m.errChan <- err:
				default:
				}
			}
		}(worker, workerCtx, name)
	}

	return nil
}

// StopWorker 停止指定协程
func (m *Manager) StopWorker(name string) error {
	m.mu.Lock()
	worker, exists := m.workers[name]
	cancel, hasCancel := m.workerContexts[name]
	m.mu.Unlock()

	if !exists {
		return ErrWorkerNotFound
	}

	if hasCancel {
		cancel()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), m.shutdownTimeout)
	defer shutdownCancel()
	worker.Stop(shutdownCtx)

	return nil
}

// OnStartup 注册启动钩子
func (m *Manager) OnStartup(fn HookFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks.onStartup = append(m.hooks.onStartup, fn)
}

// OnWorkerStart 注册协程启动钩子
func (m *Manager) OnWorkerStart(fn WorkerHookFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks.onWorkerStart = append(m.hooks.onWorkerStart, fn)
}

// OnWorkerExit 注册协程退出钩子
func (m *Manager) OnWorkerExit(fn WorkerHookFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks.onWorkerExit = append(m.hooks.onWorkerExit, fn)
}

// OnShutdown 注册退出钩子
func (m *Manager) OnShutdown(fn HookFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks.onShutdown = append(m.hooks.onShutdown, fn)
}

// OnTimeout 注册超时钩子
func (m *Manager) OnTimeout(fn HookFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks.onTimeout = append(m.hooks.onTimeout, fn)
}

// Run 启动管理器并等待退出
func (m *Manager) Run() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return ErrAlreadyRunning
	}
	m.running = true
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(m.rootCtx)
	m.cancel = cancel

	// 调用启动钩子
	if err := m.hooks.callStartup(ctx); err != nil {
		return err
	}

	// 启动所有协程
	m.startWorkers(ctx)

	// 监听信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, m.signals...)

	// 等待退出信号或协程错误
	select {
	case <-sigChan:
		// 收到退出信号
	case err := <-m.errChan:
		// 协程错误
		if err != nil {
			cancel()
			return err
		}
	case <-ctx.Done():
		// 上下文取消
	}

	// 优雅退出
	return m.shutdown()
}

// Shutdown 手动触发退出
func (m *Manager) Shutdown() error {
	if m.cancel != nil {
		m.cancel()
	}
	return m.shutdown()
}

// startWorkers 启动所有协程
func (m *Manager) startWorkers(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, name := range m.workerOrder {
		worker := m.workers[name]
		workerCtx, workerCancel := context.WithCancel(ctx)
		m.workerContexts[name] = workerCancel
		m.wg.Add(1)

		go func(w *Worker, wCtx context.Context, wName string) {
			defer func() {
				m.wg.Done()
				m.mu.Lock()
				delete(m.workerContexts, wName)
				delete(m.workers, wName)
				for i, n := range m.workerOrder {
					if n == wName {
						m.workerOrder = append(m.workerOrder[:i], m.workerOrder[i+1:]...)
						break
					}
				}
				m.mu.Unlock()
			}()

			m.hooks.callWorkerStart(w.Name(), nil)
			err := w.Run(wCtx)
			m.hooks.callWorkerExit(w.Name(), err)

			if err != nil && err != context.Canceled {
				select {
				case m.errChan <- err:
				default:
				}
			}
		}(worker, workerCtx, name)
	}
}

// shutdown 执行退出流程
func (m *Manager) shutdown() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), m.shutdownTimeout)
	defer cancel()

	// 取消所有协程的 context
	m.mu.Lock()
	for _, cancelFunc := range m.workerContexts {
		cancelFunc()
	}
	m.mu.Unlock()

	// 调用停止函数（LIFO顺序）
	m.mu.RLock()
	for i := len(m.workerOrder) - 1; i >= 0; i-- {
		name := m.workerOrder[i]
		worker := m.workers[name]
		worker.Stop(shutdownCtx)
	}
	m.mu.RUnlock()

	// 等待所有协程退出
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有协程正常退出
	case <-shutdownCtx.Done():
		// 超时
		m.hooks.callTimeout(shutdownCtx)
		return ErrShutdownTimeout
	}

	// 调用退出钩子
	return m.hooks.callShutdown(shutdownCtx)
}
