package taskpool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestTaskPool_Submit(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(2), WithMaxWorkers(5))
	defer func() { _ = pool.ShutdownNow() }()

	executed := atomic.Bool{}
	future := pool.Submit(func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	result := <-future.Wait()
	if result.Err != nil {
		t.Errorf("Task failed: %v", result.Err)
	}

	if !executed.Load() {
		t.Error("Task was not executed")
	}
}

func TestTaskPool_SubmitAndWait(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(2))
	defer func() { _ = pool.ShutdownNow() }()

	ctx := context.Background()
	result, err := pool.SubmitAndWait(ctx, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("SubmitAndWait failed: %v", err)
	}

	if result.Duration < 50*time.Millisecond {
		t.Errorf("Duration = %v, want >= 50ms", result.Duration)
	}
}

func TestTaskPool_Timeout(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(2))
	defer func() { _ = pool.ShutdownNow() }()

	future := pool.Submit(func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}, WithTimeout(50*time.Millisecond))

	result := <-future.Wait()
	if result.Err != ErrTimeout {
		t.Errorf("Expected timeout error, got: %v", result.Err)
	}
}

func TestTaskPool_Priority(t *testing.T) {
	pool := New(
		WithQueueSize(100),
		WithMinWorkers(1),
		WithMaxWorkers(1),
		WithPriorityQueue(true),
	)
	defer func() { _ = pool.ShutdownNow() }()

	time.Sleep(100 * time.Millisecond)

	results := make([]int, 0, 3)
	var mu atomic.Value
	mu.Store(&results)

	pool.Submit(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	pool.Submit(func(ctx context.Context) error {
		r := mu.Load().(*[]int)
		*r = append(*r, 10)
		return nil
	}, WithPriority(10))

	pool.Submit(func(ctx context.Context) error {
		r := mu.Load().(*[]int)
		*r = append(*r, 90)
		return nil
	}, WithPriority(90))

	pool.Submit(func(ctx context.Context) error {
		r := mu.Load().(*[]int)
		*r = append(*r, 50)
		return nil
	}, WithPriority(50))

	time.Sleep(500 * time.Millisecond)

	finalResults := mu.Load().(*[]int)
	if len(*finalResults) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(*finalResults))
	}

	if (*finalResults)[0] != 90 {
		t.Errorf("First task priority = %d, want 90", (*finalResults)[0])
	}
}

func TestTaskPool_Panic(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(2))
	defer func() { _ = pool.ShutdownNow() }()

	panicCaught := false
	pool.onTaskPanic = func(taskID string, recovered interface{}) {
		panicCaught = true
	}

	future := pool.Submit(func(ctx context.Context) error {
		panic("test panic")
	})

	result := <-future.Wait()
	if result.Panic == nil {
		t.Error("Expected panic to be captured")
	}

	if !panicCaught {
		t.Error("Panic handler was not called")
	}
}

func TestTaskPool_BatchSubmit(t *testing.T) {
	pool := New(WithQueueSize(100), WithMinWorkers(5))
	defer func() { _ = pool.ShutdownNow() }()

	count := atomic.Int32{}
	fns := make([]TaskFunc, 10)
	for i := range fns {
		fns[i] = func(ctx context.Context) error {
			count.Add(1)
			return nil
		}
	}

	futures := pool.BatchSubmit(fns)

	for _, future := range futures {
		<-future.Wait()
	}

	if count.Load() != 10 {
		t.Errorf("Executed count = %d, want 10", count.Load())
	}
}

func TestTaskPool_Resize(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(2), WithMaxWorkers(10))
	defer func() { _ = pool.ShutdownNow() }()

	if pool.GetWorkerCount() != 2 {
		t.Errorf("Initial worker count = %d, want 2", pool.GetWorkerCount())
	}

	err := pool.Resize(5)
	if err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if pool.GetWorkerCount() != 5 {
		t.Errorf("Worker count after resize = %d, want 5", pool.GetWorkerCount())
	}
}

func TestTaskPool_AutoScale(t *testing.T) {
	pool := New(
		WithQueueSize(100),
		WithMinWorkers(2),
		WithMaxWorkers(10),
		WithAutoScale(true),
		WithScaleInterval(100*time.Millisecond),
	)
	defer func() { _ = pool.ShutdownNow() }()

	scaled := false
	pool.onWorkerScale = func(oldCount, newCount int) {
		scaled = true
		t.Logf("Scaled from %d to %d workers", oldCount, newCount)
	}

	for i := 0; i < 90; i++ {
		pool.Submit(func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	}

	time.Sleep(500 * time.Millisecond)

	if !scaled {
		t.Error("Auto-scale was not triggered")
	}
}

func TestTaskPool_AutoScaleDown(t *testing.T) {
	pool := New(
		WithQueueSize(100),
		WithMinWorkers(2),
		WithMaxWorkers(10),
		WithAutoScale(true),
		WithScaleInterval(100*time.Millisecond),
	)
	defer func() { _ = pool.ShutdownNow() }()

	scaledUp := false
	scaledDown := false
	pool.onWorkerScale = func(oldCount, newCount int) {
		t.Logf("Worker count changed: %d -> %d (queue: %d, running: %d)",
			oldCount, newCount, pool.queue.Len(), pool.runningTasks.Load())
		if newCount > oldCount {
			scaledUp = true
		}
		if newCount < oldCount {
			scaledDown = true
		}
	}

	// Phase 1: Submit many tasks to trigger scale-up
	for i := 0; i < 90; i++ {
		pool.Submit(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	// Wait for scale-up to happen
	time.Sleep(200 * time.Millisecond)

	if !scaledUp {
		t.Log("Warning: Scale-up was not triggered, but continuing test")
	}

	// Phase 2: Wait for all tasks to complete and workers to become idle
	time.Sleep(1500 * time.Millisecond)

	if !scaledDown {
		t.Errorf("Auto-scale down was not triggered (workers: %d)", pool.GetWorkerCount())
	}
}

func TestTaskPool_Shutdown(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(2))

	count := atomic.Int32{}
	for i := 0; i < 5; i++ {
		pool.Submit(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			count.Add(1)
			return nil
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := pool.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	if count.Load() != 5 {
		t.Errorf("Completed tasks = %d, want 5", count.Load())
	}
}

func TestTaskPool_Metrics(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(2))
	defer func() { _ = pool.ShutdownNow() }()

	for i := 0; i < 5; i++ {
		pool.Submit(func(ctx context.Context) error {
			return nil
		})
	}

	pool.Submit(func(ctx context.Context) error {
		return errors.New("test error")
	})

	time.Sleep(200 * time.Millisecond)

	metrics := pool.GetMetrics()
	if metrics.TotalSubmitted != 6 {
		t.Errorf("TotalSubmitted = %d, want 6", metrics.TotalSubmitted)
	}

	if metrics.TotalCompleted != 6 {
		t.Errorf("TotalCompleted = %d, want 6", metrics.TotalCompleted)
	}

	if metrics.TotalFailed != 1 {
		t.Errorf("TotalFailed = %d, want 1", metrics.TotalFailed)
	}
}

func TestTaskPool_Options(t *testing.T) {
	pool := New(
		WithScaleStrategy(NewDefaultScaleStrategy(10)),
		WithStarvationPrevention(5),
		WithDefaultTimeout(5*time.Second),
		WithPanicHandler(func(taskID string, r interface{}) {}),
		WithOnWorkerScale(func(old, new int) {}),
		WithOnTaskStart(func(taskID string) {}),
		WithOnTaskComplete(func(taskID string, duration time.Duration, err error) {}),
		WithOnTaskTimeout(func(taskID string) {}),
		WithOnTaskPanic(func(taskID string, v interface{}) {}),
		WithOnShutdown(func(metrics *MetricsSnapshot) {}),
	)
	defer func() { _ = pool.ShutdownNow() }()

	if pool == nil {
		t.Fatal("Pool should not be nil")
	}
}

func TestTaskPool_SubmitAsync(t *testing.T) {
	pool := New(WithMinWorkers(2))
	defer func() { _ = pool.ShutdownNow() }()

	executed := atomic.Bool{}
	_ = pool.SubmitAsync(func(ctx context.Context) error {
		executed.Store(true)
		return nil
	})

	time.Sleep(100 * time.Millisecond)
	if !executed.Load() {
		t.Error("Async task was not executed")
	}
}

func TestTaskPool_GetQueueLength(t *testing.T) {
	pool := New(WithQueueSize(10), WithMinWorkers(1))
	defer func() { _ = pool.ShutdownNow() }()

	qLen := pool.GetQueueLength()
	if qLen < 0 {
		t.Errorf("Queue length should be non-negative, got %d", qLen)
	}
}

func TestTask_IsDone(t *testing.T) {
	pool := New(WithMinWorkers(1))
	defer func() { _ = pool.ShutdownNow() }()

	future := pool.Submit(func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	if future.IsDone() {
		t.Error("Task should not be done immediately")
	}

	<-future.Wait()
	if !future.IsDone() {
		t.Error("Task should be done after waiting")
	}
}

func TestTask_GetResult(t *testing.T) {
	pool := New(WithMinWorkers(1))
	defer func() { _ = pool.ShutdownNow() }()

	future := pool.Submit(func(ctx context.Context) error {
		return errors.New("test error")
	})

	<-future.Wait()
	result, _ := future.GetResult(100 * time.Millisecond)
	if result.Err == nil {
		t.Error("Expected error in result")
	}
}

func TestTask_WithTaskID(t *testing.T) {
	pool := New(WithMinWorkers(1))
	defer func() { _ = pool.ShutdownNow() }()

	future := pool.Submit(func(ctx context.Context) error {
		return nil
	}, WithTaskID("test-task-123"))

	result := <-future.Wait()
	if result.TaskID != "test-task-123" {
		t.Errorf("TaskID = %s, want test-task-123", result.TaskID)
	}
}

func TestQueue_Cap(t *testing.T) {
	queue := NewRingQueue(10)
	defer queue.Close()

	if queue.Cap() != 10 {
		t.Errorf("Cap = %d, want 10", queue.Cap())
	}
}

func TestScaleStrategy_ScaleDownCount(t *testing.T) {
	strategy := NewDefaultScaleStrategy(10)
	count := strategy.ScaleDownCount(10, 2)
	if count < 0 {
		t.Errorf("ScaleDownCount should be non-negative, got %d", count)
	}
}
