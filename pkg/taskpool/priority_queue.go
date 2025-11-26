package taskpool

import (
	"sync"
	"sync/atomic"
)

// PriorityRingQueue 优先级环形队列（多级桶实现）
type PriorityRingQueue struct {
	buckets       []*RingQueue // 优先级桶（10个桶，每桶覆盖10个优先级）
	bucketCount   int          // 桶数量
	priorityRange int          // 每个桶的优先级范围
	consumeCount  atomic.Int64 // 消费计数（用于防饥饿）
	starvationN   int          // 每N次强制消费低优先级
	mu            sync.RWMutex
	notEmpty      *sync.Cond
	closed        bool
	totalSize     atomic.Int32
	capacity      int
}

// NewPriorityRingQueue 创建优先级环形队列
func NewPriorityRingQueue(capacity, starvationN int) *PriorityRingQueue {
	bucketCount := 10
	bucketCapacity := capacity / bucketCount
	if bucketCapacity < 10 {
		bucketCapacity = 10
	}

	pq := &PriorityRingQueue{
		buckets:       make([]*RingQueue, bucketCount),
		bucketCount:   bucketCount,
		priorityRange: 10,
		starvationN:   starvationN,
		capacity:      capacity,
	}

	for i := 0; i < bucketCount; i++ {
		pq.buckets[i] = NewRingQueue(bucketCapacity)
	}

	pq.notEmpty = sync.NewCond(&pq.mu)
	return pq
}

// Push 入队
func (pq *PriorityRingQueue) Push(task *Task, blocking bool) error {
	pq.mu.Lock()
	if pq.closed {
		pq.mu.Unlock()
		return ErrQueueClosed
	}
	pq.mu.Unlock()

	bucketIdx := task.Priority / pq.priorityRange
	if bucketIdx >= pq.bucketCount {
		bucketIdx = pq.bucketCount - 1
	}

	err := pq.buckets[bucketIdx].Push(task, blocking)
	if err == nil {
		pq.totalSize.Add(1)
		pq.notEmpty.Signal()
	}
	return err
}

// Pop 出队（按优先级）
func (pq *PriorityRingQueue) Pop(blocking bool) (*Task, error) {
	for {
		pq.mu.Lock()
		if pq.closed && pq.totalSize.Load() == 0 {
			pq.mu.Unlock()
			return nil, ErrQueueClosed
		}

		if pq.totalSize.Load() == 0 {
			if !blocking {
				pq.mu.Unlock()
				return nil, nil
			}
			pq.notEmpty.Wait()
			pq.mu.Unlock()
			continue
		}
		pq.mu.Unlock()

		count := pq.consumeCount.Add(1)
		forceLowPriority := pq.starvationN > 0 && count%int64(pq.starvationN) == 0

		if forceLowPriority {
			for i := 0; i < pq.bucketCount; i++ {
				task, err := pq.buckets[i].Pop(false)
				if err == nil && task != nil {
					pq.totalSize.Add(-1)
					return task, nil
				}
			}
		} else {
			for i := pq.bucketCount - 1; i >= 0; i-- {
				task, err := pq.buckets[i].Pop(false)
				if err == nil && task != nil {
					pq.totalSize.Add(-1)
					return task, nil
				}
			}
		}

		if !blocking {
			return nil, nil
		}
	}
}

// BatchPush 批量入队
func (pq *PriorityRingQueue) BatchPush(tasks []*Task) error {
	pq.mu.Lock()
	if pq.closed {
		pq.mu.Unlock()
		return ErrQueueClosed
	}
	pq.mu.Unlock()

	for _, task := range tasks {
		if err := pq.Push(task, true); err != nil {
			return err
		}
	}

	pq.notEmpty.Broadcast()
	return nil
}

// Len 队列长度
func (pq *PriorityRingQueue) Len() int {
	return int(pq.totalSize.Load())
}

// Cap 队列容量
func (pq *PriorityRingQueue) Cap() int {
	return pq.capacity
}

// Close 关闭队列
func (pq *PriorityRingQueue) Close() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if !pq.closed {
		pq.closed = true
		for _, bucket := range pq.buckets {
			bucket.Close()
		}
		pq.notEmpty.Broadcast()
	}
}
