package taskpool

import (
	"context"
	"fmt"
	"time"
)

// Worker 工作协程
type Worker struct {
	id       int
	pool     *TaskPool
	stopChan chan struct{}
}

// newWorker 创建工作协程
func newWorker(id int, pool *TaskPool) *Worker {
	return &Worker{
		id:       id,
		pool:     pool,
		stopChan: make(chan struct{}),
	}
}

// run 运行工作协程
func (w *Worker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		default:
			task, err := w.pool.queue.Pop(true)
			if err != nil {
				if err == ErrQueueClosed {
					return
				}
				continue
			}

			if task == nil {
				continue
			}

			w.executeTask(ctx, task)
		}
	}
}

// stop 停止工作协程
func (w *Worker) stop() {
	close(w.stopChan)
}

// executeTask 执行任务
func (w *Worker) executeTask(ctx context.Context, task *Task) {
	w.pool.runningTasks.Add(1)
	defer w.pool.runningTasks.Add(-1)

	result := &TaskResult{
		TaskID:    task.ID,
		StartTime: time.Now(),
	}

	if w.pool.onTaskStart != nil {
		w.pool.onTaskStart(task.ID)
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		if w.pool.onTaskComplete != nil {
			w.pool.onTaskComplete(task.ID, result.Duration, result.Err)
		}

		w.pool.metrics.recordTaskComplete(result)

		if task.future != nil {
			task.future.complete(result)
		}
	}()

	taskCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	done := make(chan error, 1)
	panicChan := make(chan interface{}, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicChan <- r
			}
		}()
		done <- task.Fn(taskCtx)
	}()

	select {
	case err := <-done:
		result.Err = err
	case r := <-panicChan:
		result.Panic = r
		result.Err = fmt.Errorf("%w: %v", ErrTaskPanic, r)
		if w.pool.onTaskPanic != nil {
			w.pool.onTaskPanic(task.ID, r)
		}
		if w.pool.panicHandler != nil {
			w.pool.panicHandler(task.ID, r)
		}
	case <-taskCtx.Done():
		result.Err = ErrTimeout
		if w.pool.onTaskTimeout != nil {
			w.pool.onTaskTimeout(task.ID)
		}
	}
}
