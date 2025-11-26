package main

import (
	"context"
	"fmt"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/lifecycle"
	"github.com/junbin-yang/go-kitbox/pkg/taskpool"
)

func main() {
	fmt.Println("=== TaskPool 示例 ===\n")

	// 示例1: 基础使用
	basicExample()

	// 示例2: 优先级队列
	priorityExample()

	// 示例3: 批量提交
	batchExample()

	// 示例4: 与lifecycle集成
	lifecycleIntegrationExample()
}

// basicExample 基础使用示例
func basicExample() {
	fmt.Println("1. 基础使用")
	fmt.Println("-------------------")

	pool := taskpool.New(
		taskpool.WithQueueSize(100),
		taskpool.WithMinWorkers(5),
		taskpool.WithMaxWorkers(20),
		taskpool.WithOnTaskComplete(func(taskID string, duration time.Duration, err error) {
			fmt.Printf("  → 任务 %s 完成，耗时: %v\n", taskID, duration)
		}),
	)
	defer pool.ShutdownNow()

	// 异步提交
	future := pool.Submit(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}, taskpool.WithTaskID("task-1"))

	// 等待结果
	result := <-future.Wait()
	fmt.Printf("  → 任务结果: 耗时 %v\n", result.Duration)

	// 同步提交
	ctx := context.Background()
	result2, _ := pool.SubmitAndWait(ctx, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}, taskpool.WithTaskID("task-2"))

	fmt.Printf("  → 同步任务完成，耗时: %v\n\n", result2.Duration)
}

// priorityExample 优先级队列示例
func priorityExample() {
	fmt.Println("2. 优先级队列")
	fmt.Println("-------------------")

	pool := taskpool.New(
		taskpool.WithQueueSize(100),
		taskpool.WithMinWorkers(1), // 单个worker便于观察优先级
		taskpool.WithMaxWorkers(1),
		taskpool.WithPriorityQueue(true),
		taskpool.WithStarvationPrevention(5),
	)
	defer pool.ShutdownNow()

	// 先提交一个长任务占住worker
	pool.Submit(func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	time.Sleep(50 * time.Millisecond)

	// 提交不同优先级的任务
	pool.Submit(func(ctx context.Context) error {
		fmt.Println("  → 低优先级任务执行")
		return nil
	}, taskpool.WithPriority(10))

	pool.Submit(func(ctx context.Context) error {
		fmt.Println("  → 高优先级任务执行")
		return nil
	}, taskpool.WithPriority(90))

	pool.Submit(func(ctx context.Context) error {
		fmt.Println("  → 中优先级任务执行")
		return nil
	}, taskpool.WithPriority(50))

	time.Sleep(500 * time.Millisecond)
	fmt.Println()
}

// batchExample 批量提交示例
func batchExample() {
	fmt.Println("3. 批量提交")
	fmt.Println("-------------------")

	pool := taskpool.New(
		taskpool.WithQueueSize(100),
		taskpool.WithMinWorkers(5),
	)
	defer pool.ShutdownNow()

	// 批量提交任务
	tasks := make([]taskpool.TaskFunc, 10)
	for i := range tasks {
		idx := i
		tasks[i] = func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("  → 任务 %d 完成\n", idx)
			return nil
		}
	}

	futures := pool.BatchSubmit(tasks)

	// 等待所有任务完成
	for _, future := range futures {
		<-future.Wait()
	}

	fmt.Println()
}

// lifecycleIntegrationExample 与lifecycle集成示例
func lifecycleIntegrationExample() {
	fmt.Println("4. 与Lifecycle集成")
	fmt.Println("-------------------")

	lm := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(5 * time.Second),
	)

	pool := taskpool.New(
		taskpool.WithQueueSize(100),
		taskpool.WithMinWorkers(5),
		taskpool.WithMaxWorkers(20),
		taskpool.WithAutoScale(true),
		taskpool.WithOnShutdown(func(metrics *taskpool.MetricsSnapshot) {
			fmt.Printf("  → TaskPool 关闭统计:\n")
			fmt.Printf("     总提交: %d\n", metrics.TotalSubmitted)
			fmt.Printf("     总完成: %d\n", metrics.TotalCompleted)
			fmt.Printf("     成功率: %.2f%%\n", metrics.SuccessRate)
			fmt.Printf("     平均耗时: %v\n", metrics.AvgExecTime)
		}),
	)

	// 将taskpool作为worker添加到lifecycle
	lm.AddWorker("taskpool",
		func(ctx context.Context) error {
			fmt.Println("  → TaskPool 运行中...")
			<-ctx.Done()
			fmt.Println("  → 正在关闭 TaskPool...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return pool.Shutdown(shutdownCtx)
		},
	)

	// 添加任务提交协程
	lm.AddWorker("task-submitter",
		func(ctx context.Context) error {
			for i := 0; i < 10; i++ {
				select {
				case <-ctx.Done():
					return nil
				default:
					pool.Submit(func(ctx context.Context) error {
						time.Sleep(100 * time.Millisecond)
						return nil
					})
					time.Sleep(50 * time.Millisecond)
				}
			}
			return nil
		},
	)

	// 模拟运行一段时间后退出
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Println("\n  → 触发退出...")
		lm.Shutdown()
	}()

	if err := lm.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	fmt.Println("\n完成!")
}
