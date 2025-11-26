# TaskPool - 高性能任务协程池

提供高并发任务处理能力，支持优先级队列、动态扩缩容、超时控制、指标统计等功能。

## 特性

- **环形队列** - 手动实现线程安全的环形缓冲区，降低 GC 压力
- **优先级队列** - 可选的多级桶优先级队列，支持防饥饿机制
- **动态扩缩容** - 自动或手动调整工作协程数量，适应负载变化
- **任务超时** - 每个任务可设置独立超时时间
- **Panic 恢复** - 自动捕获任务执行中的 panic，防止工作协程崩溃
- **指标统计** - 实时统计任务执行情况、成功率、平均耗时等
- **优雅关闭** - 支持等待队列中任务执行完毕
- **多种提交方式** - 同步、异步、批量提交

## 文件结构

```
pkg/taskpool/
├── errors.go              # 错误定义
├── task.go                # 任务和Future定义
├── options.go             # 配置选项
├── queue.go               # 环形队列实现
├── priority_queue.go      # 优先级队列实现
├── scale.go               # 扩缩容策略
├── worker.go              # 工作协程
├── metrics.go             # 指标统计
├── taskpool.go            # 核心协程池
├── queue_test.go          # 队列测试
├── priority_queue_test.go # 优先级队列测试
├── taskpool_test.go       # 核心功能测试
├── benchmark_test.go      # 性能基准测试
└── README.md              # 文档

examples/taskpool_example/
└── main.go                # 使用示例
```

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/taskpool
```

## 快速开始

### 默认配置

`taskpool.New()` 不传任何参数时使用以下默认配置：

- **队列大小**: 1000
- **最小工作协程数**: 5
- **最大工作协程数**: 50
- **优先级队列**: 关闭
- **自动扩缩容**: 关闭
- **扩缩容检查间隔**: 5秒
- **防饥饿参数**: 每10次高优先级后消费1次低优先级
- **默认任务超时**: 无超时

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/junbin-yang/go-kitbox/pkg/taskpool"
)

func main() {
    // 使用默认配置创建协程池
    pool := taskpool.New()
    defer pool.ShutdownNow()

    // 或自定义配置
    pool := taskpool.New(
        taskpool.WithQueueSize(1000),
        taskpool.WithMinWorkers(10),
        taskpool.WithMaxWorkers(100),
    )
    defer pool.ShutdownNow()

    // 提交任务
    future := pool.Submit(func(ctx context.Context) error {
        // 业务逻辑
        fmt.Println("Task executed")
        return nil
    })

    // 等待结果
    result := <-future.Wait()
    if result.Err != nil {
        fmt.Printf("Task failed: %v\n", result.Err)
    }
}
```

## 核心功能

### 1. 同步/异步提交

```go
// 异步提交（返回Future）
future := pool.Submit(func(ctx context.Context) error {
    return doWork()
})

// 同步等待结果
ctx := context.Background()
result, err := pool.SubmitAndWait(ctx, func(ctx context.Context) error {
    return doWork()
})

// 完全异步（不关心结果）
err := pool.SubmitAsync(func(ctx context.Context) error {
    return doWork()
})
```

### 2. 优先级队列

```go
pool := taskpool.New(
    taskpool.WithPriorityQueue(true),
    taskpool.WithStarvationPrevention(10), // 每10次高优先级后消费1次低优先级
)

// 提交高优先级任务
pool.Submit(urgentTask, taskpool.WithPriority(90))

// 提交普通任务
pool.Submit(normalTask, taskpool.WithPriority(50))

// 提交低优先级任务
pool.Submit(lowTask, taskpool.WithPriority(10))
```

### 3. 任务超时

```go
// 设置默认超时
pool := taskpool.New(
    taskpool.WithDefaultTimeout(5 * time.Second),
)

// 为单个任务设置超时
pool.Submit(func(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(10 * time.Second):
        return nil
    }
}, taskpool.WithTimeout(3*time.Second))
```

### 4. 动态扩缩容

```go
// 启用自动扩缩容
pool := taskpool.New(
    taskpool.WithMinWorkers(5),
    taskpool.WithMaxWorkers(50),
    taskpool.WithAutoScale(true),
    taskpool.WithScaleInterval(5*time.Second),
    taskpool.WithOnWorkerScale(func(oldCount, newCount int) {
        fmt.Printf("Workers scaled: %d -> %d\n", oldCount, newCount)
    }),
)

// 手动调整协程数
pool.Resize(20)
```

### 5. 批量提交

```go
tasks := []taskpool.TaskFunc{
    func(ctx context.Context) error { return task1() },
    func(ctx context.Context) error { return task2() },
    func(ctx context.Context) error { return task3() },
}

futures := pool.BatchSubmit(tasks)

// 等待所有任务完成
for _, future := range futures {
    result := <-future.Wait()
    fmt.Printf("Task %s completed\n", result.TaskID)
}
```

### 6. 钩子函数

```go
pool := taskpool.New(
    taskpool.WithOnTaskStart(func(taskID string) {
        fmt.Printf("Task %s started\n", taskID)
    }),
    taskpool.WithOnTaskComplete(func(taskID string, duration time.Duration, err error) {
        fmt.Printf("Task %s completed in %v\n", taskID, duration)
    }),
    taskpool.WithOnTaskPanic(func(taskID string, recovered interface{}) {
        fmt.Printf("Task %s panicked: %v\n", taskID, recovered)
    }),
    taskpool.WithOnShutdown(func(metrics *taskpool.MetricsSnapshot) {
        fmt.Printf("Pool shutdown. Total tasks: %d, Success rate: %.2f%%\n",
            metrics.TotalCompleted, metrics.SuccessRate)
    }),
)
```

### 7. 指标统计

```go
metrics := pool.GetMetrics()

fmt.Printf("Total submitted: %d\n", metrics.TotalSubmitted)
fmt.Printf("Total completed: %d\n", metrics.TotalCompleted)
fmt.Printf("Success rate: %.2f%%\n", metrics.SuccessRate)
fmt.Printf("Avg execution time: %v\n", metrics.AvgExecTime)
fmt.Printf("Current queue length: %d\n", metrics.CurrentQueue)
fmt.Printf("Running tasks: %d\n", metrics.RunningTasks)
fmt.Printf("Active workers: %d\n", metrics.ActiveWorkers)
```

### 8. 优雅关闭

```go
// 等待所有任务完成后关闭
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := pool.Shutdown(ctx); err != nil {
    fmt.Printf("Shutdown timeout: %v\n", err)
}

// 立即关闭（丢弃队列中的任务）
pool.ShutdownNow()
```

## 与 Lifecycle 集成

TaskPool 可以无缝集成到 Lifecycle 管理器中，实现统一的生命周期管理：

```go
import (
    "context"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/lifecycle"
    "github.com/junbin-yang/go-kitbox/pkg/taskpool"
)

func main() {
    lm := lifecycle.NewManager(
        lifecycle.WithShutdownTimeout(5 * time.Second),
    )

    pool := taskpool.New(
        taskpool.WithQueueSize(1000),
        taskpool.WithMinWorkers(10),
        taskpool.WithMaxWorkers(100),
        taskpool.WithAutoScale(true),
    )

    // 将taskpool作为worker添加到lifecycle
    lm.AddWorker("taskpool",
        func(ctx context.Context) error {
            <-ctx.Done()
            // 优雅关闭，等待任务完成
            shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()
            return pool.Shutdown(shutdownCtx)
        },
    )

    lm.Run()
}
```

## API 参考

### 配置选项

| 选项 | 说明 |
|------|------|
| `WithQueueSize(size)` | 设置队列大小 |
| `WithMinWorkers(n)` | 设置最小工作协程数 |
| `WithMaxWorkers(n)` | 设置最大工作协程数 |
| `WithPriorityQueue(enable)` | 启用优先级队列 |
| `WithAutoScale(enable)` | 启用自动扩缩容 |
| `WithScaleInterval(interval)` | 设置扩缩容检查间隔 |
| `WithStarvationPrevention(n)` | 设置防饥饿参数 |
| `WithDefaultTimeout(timeout)` | 设置默认任务超时 |
| `WithOnWorkerScale(fn)` | 设置扩缩容钩子 |
| `WithOnTaskStart(fn)` | 设置任务开始钩子 |
| `WithOnTaskComplete(fn)` | 设置任务完成钩子 |
| `WithOnTaskTimeout(fn)` | 设置任务超时钩子 |
| `WithOnTaskPanic(fn)` | 设置任务panic钩子 |
| `WithOnShutdown(fn)` | 设置关闭钩子 |

### 任务选项

| 选项 | 说明 |
|------|------|
| `WithTaskID(id)` | 设置任务ID |
| `WithPriority(p)` | 设置任务优先级（0-100） |
| `WithTimeout(d)` | 设置任务超时时间 |

## 最佳实践

1. **合理设置队列大小** - 根据任务提交速率和处理速度设置
2. **使用优先级队列** - 对于有明确优先级需求的场景
3. **启用自动扩缩容** - 应对负载波动
4. **设置任务超时** - 防止任务长时间阻塞
5. **监控指标** - 定期获取指标进行性能分析
6. **优雅关闭** - 确保任务完整执行

## 许可证

MIT License
