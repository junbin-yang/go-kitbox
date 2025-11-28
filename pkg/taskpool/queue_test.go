package taskpool

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRingQueue_PushPop(t *testing.T) {
	q := NewRingQueue(10)
	defer q.Close()

	task := newTask(func(ctx context.Context) error { return nil })

	if err := q.Push(task, false); err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Len() = %d, want 1", q.Len())
	}

	popped, err := q.Pop(false)
	if err != nil {
		t.Fatalf("Pop failed: %v", err)
	}

	if popped.ID != task.ID {
		t.Errorf("Popped task ID = %s, want %s", popped.ID, task.ID)
	}

	if q.Len() != 0 {
		t.Errorf("Len() = %d, want 0", q.Len())
	}
}

func TestRingQueue_Full(t *testing.T) {
	q := NewRingQueue(2)
	defer q.Close()

	task1 := newTask(func(ctx context.Context) error { return nil })
	task2 := newTask(func(ctx context.Context) error { return nil })
	task3 := newTask(func(ctx context.Context) error { return nil })

	_ = q.Push(task1, false)
	_ = q.Push(task2, false)

	err := q.Push(task3, false)
	if err != ErrQueueFull {
		t.Errorf("Push to full queue: got %v, want ErrQueueFull", err)
	}
}

func TestRingQueue_BlockingPush(t *testing.T) {
	q := NewRingQueue(1)
	defer q.Close()

	task1 := newTask(func(ctx context.Context) error { return nil })
	task2 := newTask(func(ctx context.Context) error { return nil })

	_ = q.Push(task1, false)

	done := make(chan bool)
	go func() {
		_ = q.Push(task2, true)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	_, _ = q.Pop(false)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Blocking push timeout")
	}
}

func TestRingQueue_BlockingPop(t *testing.T) {
	q := NewRingQueue(10)
	defer q.Close()

	task := newTask(func(ctx context.Context) error { return nil })

	done := make(chan *Task)
	go func() {
		popped, _ := q.Pop(true)
		done <- popped
	}()

	time.Sleep(50 * time.Millisecond)
	_ = q.Push(task, false)

	select {
	case popped := <-done:
		if popped.ID != task.ID {
			t.Errorf("Popped task ID = %s, want %s", popped.ID, task.ID)
		}
	case <-time.After(time.Second):
		t.Error("Blocking pop timeout")
	}
}

func TestRingQueue_BatchPush(t *testing.T) {
	q := NewRingQueue(10)
	defer q.Close()

	tasks := make([]*Task, 5)
	for i := range tasks {
		tasks[i] = newTask(func(ctx context.Context) error { return nil })
	}

	if err := q.BatchPush(tasks); err != nil {
		t.Fatalf("BatchPush failed: %v", err)
	}

	if q.Len() != 5 {
		t.Errorf("Len() = %d, want 5", q.Len())
	}
}

func TestRingQueue_Concurrent(t *testing.T) {
	q := NewRingQueue(100)
	defer q.Close()

	var wg sync.WaitGroup
	pushCount := 1000
	popCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				task := newTask(func(ctx context.Context) error { return nil })
				q.Push(task, true)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, err := q.Pop(true); err == nil {
					mu.Lock()
					popCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	if popCount != pushCount {
		t.Errorf("Pop count = %d, want %d", popCount, pushCount)
	}
}

func TestRingQueue_Close(t *testing.T) {
	q := NewRingQueue(10)

	task := newTask(func(ctx context.Context) error { return nil })
	q.Push(task, false)

	q.Close()

	err := q.Push(task, false)
	if err != ErrQueueClosed {
		t.Errorf("Push after close: got %v, want ErrQueueClosed", err)
	}

	popped, err := q.Pop(false)
	if popped == nil && err != ErrQueueClosed {
		t.Errorf("Pop empty closed queue: got %v, want ErrQueueClosed", err)
	}
}
