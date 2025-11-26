package taskpool

import "time"

// Option 协程池配置选项
type Option func(*TaskPool)

// WithQueueSize 设置队列大小
func WithQueueSize(size int) Option {
	return func(p *TaskPool) {
		p.queueSize = size
	}
}

// WithMinWorkers 设置最小工作协程数
func WithMinWorkers(n int) Option {
	return func(p *TaskPool) {
		p.minWorkers = n
	}
}

// WithMaxWorkers 设置最大工作协程数
func WithMaxWorkers(n int) Option {
	return func(p *TaskPool) {
		p.maxWorkers = n
	}
}

// WithPriorityQueue 启用优先级队列
func WithPriorityQueue(enable bool) Option {
	return func(p *TaskPool) {
		p.enablePriority = enable
	}
}

// WithAutoScale 启用自动扩缩容
func WithAutoScale(enable bool) Option {
	return func(p *TaskPool) {
		p.autoScale = enable
	}
}

// WithScaleStrategy 设置扩缩容策略
func WithScaleStrategy(strategy ScaleStrategy) Option {
	return func(p *TaskPool) {
		p.scaleStrategy = strategy
	}
}

// WithScaleInterval 设置扩缩容检查间隔
func WithScaleInterval(interval time.Duration) Option {
	return func(p *TaskPool) {
		p.scaleInterval = interval
	}
}

// WithStarvationPrevention 设置防饥饿参数（每N次消费高优先级后消费1次低优先级）
func WithStarvationPrevention(n int) Option {
	return func(p *TaskPool) {
		p.starvationN = n
	}
}

// WithDefaultTimeout 设置默认任务超时时间
func WithDefaultTimeout(timeout time.Duration) Option {
	return func(p *TaskPool) {
		p.defaultTimeout = timeout
	}
}

// WithPanicHandler 设置panic处理函数
func WithPanicHandler(fn func(taskID string, r interface{})) Option {
	return func(p *TaskPool) {
		p.panicHandler = fn
	}
}

// WithOnWorkerScale 设置工作协程扩缩容钩子
func WithOnWorkerScale(fn func(oldCount, newCount int)) Option {
	return func(p *TaskPool) {
		p.onWorkerScale = fn
	}
}

// WithOnTaskStart 设置任务开始钩子
func WithOnTaskStart(fn func(taskID string)) Option {
	return func(p *TaskPool) {
		p.onTaskStart = fn
	}
}

// WithOnTaskComplete 设置任务完成钩子
func WithOnTaskComplete(fn func(taskID string, duration time.Duration, err error)) Option {
	return func(p *TaskPool) {
		p.onTaskComplete = fn
	}
}

// WithOnTaskTimeout 设置任务超时钩子
func WithOnTaskTimeout(fn func(taskID string)) Option {
	return func(p *TaskPool) {
		p.onTaskTimeout = fn
	}
}

// WithOnTaskPanic 设置任务panic钩子
func WithOnTaskPanic(fn func(taskID string, recovered interface{})) Option {
	return func(p *TaskPool) {
		p.onTaskPanic = fn
	}
}

// WithOnShutdown 设置关闭钩子（返回最终指标）
func WithOnShutdown(fn func(metrics *MetricsSnapshot)) Option {
	return func(p *TaskPool) {
		p.onShutdown = fn
	}
}

// TaskOption 任务配置选项
type TaskOption func(*Task)

// WithTaskID 设置任务ID
func WithTaskID(id string) TaskOption {
	return func(t *Task) {
		t.ID = id
	}
}

// WithPriority 设置任务优先级（0-100）
func WithPriority(p int) TaskOption {
	return func(t *Task) {
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		t.Priority = p
	}
}

// WithTimeout 设置任务超时时间
func WithTimeout(d time.Duration) TaskOption {
	return func(t *Task) {
		t.Timeout = d
	}
}
