package taskpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// TaskPool 任务协程池
type TaskPool struct {
	queue           Queue
	workers         []*Worker
	minWorkers      int
	maxWorkers      int
	currentWorkers  atomic.Int32
	runningTasks    atomic.Int32
	metrics         *Metrics
	ctx             context.Context
	cancel          context.CancelFunc
	scaleCtx        context.Context
	scaleCancel     context.CancelFunc
	wg              sync.WaitGroup
	closed          atomic.Bool
	mu              sync.RWMutex
	queueSize       int
	enablePriority  bool
	autoScale       bool
	scaleStrategy   ScaleStrategy
	scaleInterval   time.Duration
	starvationN     int
	defaultTimeout  time.Duration
	panicHandler    func(taskID string, r interface{})
	onWorkerScale   func(oldCount, newCount int)
	onTaskStart     func(taskID string)
	onTaskComplete  func(taskID string, duration time.Duration, err error)
	onTaskTimeout   func(taskID string)
	onTaskPanic     func(taskID string, recovered interface{})
	onShutdown      func(metrics *MetricsSnapshot)
}

// New 创建任务协程池
func New(opts ...Option) *TaskPool {
	p := &TaskPool{
		queueSize:      1000,
		minWorkers:     5,
		maxWorkers:     50,
		enablePriority: false,
		autoScale:      false,
		scaleInterval:  5 * time.Second,
		starvationN:    10,
		defaultTimeout: 0,
		metrics:        newMetrics(),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.enablePriority {
		p.queue = NewPriorityRingQueue(p.queueSize, p.starvationN)
	} else {
		p.queue = NewRingQueue(p.queueSize)
	}

	if p.scaleStrategy == nil {
		p.scaleStrategy = NewDefaultScaleStrategy(p.queueSize)
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.scaleCtx, p.scaleCancel = context.WithCancel(context.Background())

	for i := 0; i < p.minWorkers; i++ {
		p.addWorker()
	}

	if p.autoScale {
		p.wg.Add(1)
		go p.scaleLoop()
	}

	return p
}

// Submit 提交任务（异步，返回Future）
func (p *TaskPool) Submit(fn TaskFunc, opts ...TaskOption) *Future {
	if p.closed.Load() {
		future := newFuture("closed")
		future.complete(&TaskResult{Err: ErrPoolClosed})
		return future
	}

	task := newTask(fn, opts...)
	if task.Timeout == 0 && p.defaultTimeout > 0 {
		task.Timeout = p.defaultTimeout
	}

	task.future = newFuture(task.ID)
	p.metrics.recordSubmit()

	if err := p.queue.Push(task, true); err != nil {
		task.future.complete(&TaskResult{TaskID: task.ID, Err: err})
	}

	return task.future
}

// SubmitAndWait 提交任务并等待结果（同步）
func (p *TaskPool) SubmitAndWait(ctx context.Context, fn TaskFunc, opts ...TaskOption) (*TaskResult, error) {
	future := p.Submit(fn, opts...)

	select {
	case result := <-future.Wait():
		return result, result.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SubmitAsync 提交任务（完全异步，不关心结果）
func (p *TaskPool) SubmitAsync(fn TaskFunc, opts ...TaskOption) error {
	if p.closed.Load() {
		return ErrPoolClosed
	}

	task := newTask(fn, opts...)
	if task.Timeout == 0 && p.defaultTimeout > 0 {
		task.Timeout = p.defaultTimeout
	}

	p.metrics.recordSubmit()
	return p.queue.Push(task, false)
}

// BatchSubmit 批量提交任务
func (p *TaskPool) BatchSubmit(fns []TaskFunc, opts ...TaskOption) []*Future {
	futures := make([]*Future, len(fns))

	for i, fn := range fns {
		futures[i] = p.Submit(fn, opts...)
	}

	return futures
}

// Resize 手动调整工作协程数
func (p *TaskPool) Resize(n int) error {
	if n < p.minWorkers || n > p.maxWorkers {
		return ErrInvalidWorkerCount
	}

	current := int(p.currentWorkers.Load())
	if n > current {
		for i := 0; i < n-current; i++ {
			p.addWorker()
		}
	} else if n < current {
		for i := 0; i < current-n; i++ {
			p.removeWorker()
		}
	}

	return nil
}

// Shutdown 优雅关闭（等待所有任务完成）
func (p *TaskPool) Shutdown(ctx context.Context) error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	p.queue.Close()
	p.scaleCancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.cancel()
		if p.onShutdown != nil {
			p.onShutdown(p.GetMetrics())
		}
		return nil
	case <-ctx.Done():
		p.cancel()
		if p.onShutdown != nil {
			p.onShutdown(p.GetMetrics())
		}
		return ctx.Err()
	}
}

// ShutdownNow 立即关闭（丢弃队列中的任务）
func (p *TaskPool) ShutdownNow() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}

	p.scaleCancel()
	p.cancel()
	p.queue.Close()
	p.wg.Wait()

	if p.onShutdown != nil {
		p.onShutdown(p.GetMetrics())
	}

	return nil
}

// GetMetrics 获取指标快照
func (p *TaskPool) GetMetrics() *MetricsSnapshot {
	return p.metrics.snapshot(
		p.queue.Len(),
		int(p.runningTasks.Load()),
		int(p.currentWorkers.Load()),
	)
}

// GetWorkerCount 获取当前工作协程数
func (p *TaskPool) GetWorkerCount() int {
	return int(p.currentWorkers.Load())
}

// GetQueueLength 获取队列长度
func (p *TaskPool) GetQueueLength() int {
	return p.queue.Len()
}

// addWorker 添加工作协程
func (p *TaskPool) addWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := len(p.workers)
	worker := newWorker(id, p)
	p.workers = append(p.workers, worker)
	p.currentWorkers.Add(1)

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		worker.run(p.ctx)
	}()
}

// removeWorker 移除工作协程
func (p *TaskPool) removeWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.workers) == 0 {
		return
	}

	worker := p.workers[len(p.workers)-1]
	p.workers = p.workers[:len(p.workers)-1]
	p.currentWorkers.Add(-1)

	worker.stop()
}

// scaleLoop 自动扩缩容循环
func (p *TaskPool) scaleLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.scaleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.scaleCtx.Done():
			return
		case <-ticker.C:
			p.checkAndScale()
		}
	}
}

// checkAndScale 检查并执行扩缩容
func (p *TaskPool) checkAndScale() {
	queueLen := p.queue.Len()
	runningTasks := int(p.runningTasks.Load())
	currentWorkers := int(p.currentWorkers.Load())

	if p.scaleStrategy.ShouldScaleUp(queueLen, runningTasks, currentWorkers, p.maxWorkers) {
		count := p.scaleStrategy.ScaleUpCount(currentWorkers, p.maxWorkers)
		oldCount := currentWorkers
		for i := 0; i < count; i++ {
			p.addWorker()
		}
		newCount := int(p.currentWorkers.Load())
		if p.onWorkerScale != nil {
			p.onWorkerScale(oldCount, newCount)
		}
	} else if p.scaleStrategy.ShouldScaleDown(queueLen, runningTasks, currentWorkers, p.minWorkers) {
		count := p.scaleStrategy.ScaleDownCount(currentWorkers, p.minWorkers)
		oldCount := currentWorkers
		for i := 0; i < count; i++ {
			p.removeWorker()
		}
		newCount := int(p.currentWorkers.Load())
		if p.onWorkerScale != nil {
			p.onWorkerScale(oldCount, newCount)
		}
	}
}
