package taskpool

import (
	"context"
	"testing"
)

func TestPriorityQueue_Priority(t *testing.T) {
	pq := NewPriorityRingQueue(100, 10)
	defer pq.Close()

	lowTask := newTask(func(ctx context.Context) error { return nil }, WithPriority(10))
	midTask := newTask(func(ctx context.Context) error { return nil }, WithPriority(50))
	highTask := newTask(func(ctx context.Context) error { return nil }, WithPriority(90))

	_ = pq.Push(lowTask, false)
	_ = pq.Push(midTask, false)
	_ = pq.Push(highTask, false)

	task1, _ := pq.Pop(false)
	if task1.Priority != 90 {
		t.Errorf("First pop priority = %d, want 90", task1.Priority)
	}

	task2, _ := pq.Pop(false)
	if task2.Priority != 50 {
		t.Errorf("Second pop priority = %d, want 50", task2.Priority)
	}

	task3, _ := pq.Pop(false)
	if task3.Priority != 10 {
		t.Errorf("Third pop priority = %d, want 10", task3.Priority)
	}
}

func TestPriorityQueue_StarvationPrevention(t *testing.T) {
	pq := NewPriorityRingQueue(100, 3)
	defer pq.Close()

	for i := 0; i < 5; i++ {
		lowTask := newTask(func(ctx context.Context) error { return nil }, WithPriority(10))
		highTask := newTask(func(ctx context.Context) error { return nil }, WithPriority(90))
		_ = pq.Push(lowTask, false)
		_ = pq.Push(highTask, false)
	}

	lowCount := 0
	highCount := 0

	for i := 0; i < 10; i++ {
		task, _ := pq.Pop(false)
		if task.Priority == 10 {
			lowCount++
		} else {
			highCount++
		}
	}

	if lowCount == 0 {
		t.Error("Low priority tasks were starved")
	}

	if highCount < lowCount {
		t.Errorf("High priority count (%d) should be >= low priority count (%d)", highCount, lowCount)
	}
}

func TestPriorityQueue_Len(t *testing.T) {
	pq := NewPriorityRingQueue(100, 10)
	defer pq.Close()

	for i := 0; i < 5; i++ {
		task := newTask(func(ctx context.Context) error { return nil }, WithPriority(i*20))
		pq.Push(task, false)
	}

	if pq.Len() != 5 {
		t.Errorf("Len() = %d, want 5", pq.Len())
	}

	_, _ = pq.Pop(false)
	_, _ = pq.Pop(false)

	if pq.Len() != 3 {
		t.Errorf("Len() = %d, want 3", pq.Len())
	}
}

func TestPriorityQueue_BatchPush(t *testing.T) {
	pq := NewPriorityRingQueue(100, 10)
	defer pq.Close()

	tasks := make([]*Task, 5)
	for i := range tasks {
		tasks[i] = newTask(func(ctx context.Context) error { return nil }, WithPriority(i*20))
	}

	if err := pq.BatchPush(tasks); err != nil {
		t.Fatalf("BatchPush failed: %v", err)
	}

	if pq.Len() != 5 {
		t.Errorf("Len() = %d, want 5", pq.Len())
	}
}
