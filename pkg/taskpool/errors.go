package taskpool

import "errors"

var (
	// ErrQueueFull 队列已满
	ErrQueueFull = errors.New("taskpool: queue is full")

	// ErrQueueClosed 队列已关闭
	ErrQueueClosed = errors.New("taskpool: queue is closed")

	// ErrPoolClosed 协程池已关闭
	ErrPoolClosed = errors.New("taskpool: pool is closed")

	// ErrTimeout 操作超时
	ErrTimeout = errors.New("taskpool: operation timeout")

	// ErrInvalidWorkerCount 无效的工作协程数
	ErrInvalidWorkerCount = errors.New("taskpool: invalid worker count")

	// ErrTaskPanic 任务执行panic
	ErrTaskPanic = errors.New("taskpool: task panic")
)
