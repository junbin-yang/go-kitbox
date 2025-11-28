package taskpool

import (
	"context"
	"testing"
	"time"
)

func BenchmarkTaskPool_Submit(b *testing.B) {
	pool := New(WithQueueSize(10000), WithMinWorkers(10), WithMaxWorkers(50))
	defer func() { _ = pool.ShutdownNow() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkTaskPool_SubmitAndWait(b *testing.B) {
	pool := New(WithQueueSize(10000), WithMinWorkers(10), WithMaxWorkers(50))
	defer func() { _ = pool.ShutdownNow() }()

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.SubmitAndWait(ctx, func(ctx context.Context) error {
			return nil
		})
	}
}

func BenchmarkTaskPool_Priority(b *testing.B) {
	pool := New(
		WithQueueSize(10000),
		WithMinWorkers(10),
		WithMaxWorkers(50),
		WithPriorityQueue(true),
	)
	defer func() { _ = pool.ShutdownNow() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(func(ctx context.Context) error {
			return nil
		}, WithPriority(i%100))
	}
}

func BenchmarkTaskPool_Concurrent(b *testing.B) {
	pool := New(WithQueueSize(10000), WithMinWorkers(10), WithMaxWorkers(50))
	defer func() { _ = pool.ShutdownNow() }()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Submit(func(ctx context.Context) error {
				time.Sleep(time.Microsecond)
				return nil
			})
		}
	})
}
