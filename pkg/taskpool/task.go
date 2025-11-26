package taskpool

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// TaskFunc 任务函数类型
type TaskFunc func(context.Context) error

// Task 任务
type Task struct {
	ID       string
	Fn       TaskFunc
	Priority int           // 优先级 0-100，数值越大优先级越高
	Timeout  time.Duration // 超时时间
	SubmitAt time.Time     // 提交时间
	future   *Future       // 结果Future
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID    string
	Err       error
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Panic     interface{} // panic信息
}

// Future 异步结果
type Future struct {
	taskID   string
	resultCh chan *TaskResult
	done     atomic.Bool
	result   *TaskResult
}

// newFuture 创建Future
func newFuture(taskID string) *Future {
	return &Future{
		taskID:   taskID,
		resultCh: make(chan *TaskResult, 1),
	}
}

// Wait 等待结果（阻塞）
func (f *Future) Wait() <-chan *TaskResult {
	return f.resultCh
}

// IsDone 是否完成
func (f *Future) IsDone() bool {
	return f.done.Load()
}

// GetResult 获取结果（带超时）
func (f *Future) GetResult(timeout time.Duration) (*TaskResult, error) {
	if f.done.Load() && f.result != nil {
		return f.result, nil
	}

	select {
	case result := <-f.resultCh:
		f.result = result
		return result, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}

// complete 完成Future
func (f *Future) complete(result *TaskResult) {
	if f.done.CompareAndSwap(false, true) {
		f.result = result
		select {
		case f.resultCh <- result:
		default:
		}
		close(f.resultCh)
	}
}

// TaskID 获取任务ID
func (f *Future) TaskID() string {
	return f.taskID
}

// newTask 创建任务
func newTask(fn TaskFunc, opts ...TaskOption) *Task {
	task := &Task{
		ID:       generateTaskID(),
		Fn:       fn,
		Priority: 0,
		SubmitAt: time.Now(),
	}

	for _, opt := range opts {
		opt(task)
	}

	return task
}

// generateTaskID 生成任务ID
var taskIDCounter atomic.Uint64

func generateTaskID() string {
	return fmt.Sprintf("task-%d", taskIDCounter.Add(1))
}
