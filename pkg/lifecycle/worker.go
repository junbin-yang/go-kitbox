package lifecycle

import "context"

// RunFunc 协程运行函数
type RunFunc func(ctx context.Context) error

// StopFunc 协程停止函数
type StopFunc func(ctx context.Context) error

// Worker 协程抽象
type Worker struct {
	name     string
	runFunc  RunFunc
	stopFunc StopFunc
	err      error
}

// WorkerOption 协程配置选项
type WorkerOption func(*Worker)

// NewWorker 创建新的协程
func NewWorker(name string, runFunc RunFunc, opts ...WorkerOption) *Worker {
	w := &Worker{
		name:    name,
		runFunc: runFunc,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// WithStopFunc 设置停止函数
func WithStopFunc(stopFunc StopFunc) WorkerOption {
	return func(w *Worker) {
		w.stopFunc = stopFunc
	}
}

// Name 返回协程名称
func (w *Worker) Name() string {
	return w.name
}

// Run 运行协程
func (w *Worker) Run(ctx context.Context) error {
	w.err = w.runFunc(ctx)
	return w.err
}

// Stop 停止协程
func (w *Worker) Stop(ctx context.Context) error {
	if w.stopFunc != nil {
		return w.stopFunc(ctx)
	}
	return nil
}

// Err 返回协程错误
func (w *Worker) Err() error {
	return w.err
}
