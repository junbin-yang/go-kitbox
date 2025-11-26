package taskpool

import "sync"

// Queue 队列接口
type Queue interface {
	Push(task *Task, blocking bool) error
	Pop(blocking bool) (*Task, error)
	BatchPush(tasks []*Task) error
	Len() int
	Cap() int
	Close()
}

// RingQueue 环形队列
type RingQueue struct {
	buffer   []*Task
	head     int
	tail     int
	size     int
	capacity int
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	closed   bool
}

// NewRingQueue 创建环形队列
func NewRingQueue(capacity int) *RingQueue {
	q := &RingQueue{
		buffer:   make([]*Task, capacity),
		capacity: capacity,
	}
	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)
	return q
}

// Push 入队
func (q *RingQueue) Push(task *Task, blocking bool) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.size == q.capacity {
		if q.closed {
			return ErrQueueClosed
		}
		if !blocking {
			return ErrQueueFull
		}
		q.notFull.Wait()
	}

	if q.closed {
		return ErrQueueClosed
	}

	q.buffer[q.tail] = task
	q.tail = (q.tail + 1) % q.capacity
	q.size++

	q.notEmpty.Signal()
	return nil
}

// Pop 出队
func (q *RingQueue) Pop(blocking bool) (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.size == 0 {
		if q.closed {
			return nil, ErrQueueClosed
		}
		if !blocking {
			return nil, nil
		}
		q.notEmpty.Wait()
	}

	if q.closed && q.size == 0 {
		return nil, ErrQueueClosed
	}

	task := q.buffer[q.head]
	q.buffer[q.head] = nil
	q.head = (q.head + 1) % q.capacity
	q.size--

	q.notFull.Signal()
	return task, nil
}

// BatchPush 批量入队
func (q *RingQueue) BatchPush(tasks []*Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrQueueClosed
	}

	for _, task := range tasks {
		for q.size == q.capacity {
			if q.closed {
				return ErrQueueClosed
			}
			q.notFull.Wait()
		}

		q.buffer[q.tail] = task
		q.tail = (q.tail + 1) % q.capacity
		q.size++
	}

	q.notEmpty.Broadcast()
	return nil
}

// Len 队列长度
func (q *RingQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.size
}

// Cap 队列容量
func (q *RingQueue) Cap() int {
	return q.capacity
}

// Close 关闭队列
func (q *RingQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		q.notEmpty.Broadcast()
		q.notFull.Broadcast()
	}
}
