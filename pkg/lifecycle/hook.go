package lifecycle

import "context"

// HookFunc 钩子函数
type HookFunc func(ctx context.Context) error

// WorkerHookFunc 协程钩子函数
type WorkerHookFunc func(name string, err error)

// Hooks 钩子集合
type Hooks struct {
	onStartup     []HookFunc
	onWorkerStart []WorkerHookFunc
	onWorkerExit  []WorkerHookFunc
	onShutdown    []HookFunc
	onTimeout     []HookFunc
}

// newHooks 创建钩子集合
func newHooks() *Hooks {
	return &Hooks{
		onStartup:     make([]HookFunc, 0),
		onWorkerStart: make([]WorkerHookFunc, 0),
		onWorkerExit:  make([]WorkerHookFunc, 0),
		onShutdown:    make([]HookFunc, 0),
		onTimeout:     make([]HookFunc, 0),
	}
}

// callStartup 调用启动钩子
func (h *Hooks) callStartup(ctx context.Context) error {
	for _, fn := range h.onStartup {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

// callWorkerStart 调用协程启动钩子
func (h *Hooks) callWorkerStart(name string, err error) {
	for _, fn := range h.onWorkerStart {
		fn(name, err)
	}
}

// callWorkerExit 调用协程退出钩子
func (h *Hooks) callWorkerExit(name string, err error) {
	for _, fn := range h.onWorkerExit {
		fn(name, err)
	}
}

// callShutdown 调用退出钩子
func (h *Hooks) callShutdown(ctx context.Context) error {
	for _, fn := range h.onShutdown {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

// callTimeout 调用超时钩子
func (h *Hooks) callTimeout(ctx context.Context) error {
	for _, fn := range h.onTimeout {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}
