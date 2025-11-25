# Lifecycle - 应用生命周期管理

提供统一的应用生命周期管理，包括协程管理、信号监听、优雅退出、资源清理等功能。

## 特性

- **信号管理** - 内置退出信号监听（SIGINT/SIGTERM）和手动触发
- **协程管理** - 统一管理应用内所有协程，自动传播 Context
- **钩子系统** - 支持启动、退出、协程生命周期等钩子
- **超时控制** - 防止协程阻塞退出过程
- **优雅退出** - 按正确顺序关闭所有协程
- **错误处理** - 协程错误收集和传播
- **并发安全** - 所有操作都是线程安全的

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/lifecycle
```

## 快速开始

### 基础使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/lifecycle"
)

func main() {
    // 创建管理器
    manager := lifecycle.NewManager()

    // 添加协程
    manager.AddWorker("worker", func(ctx context.Context) error {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return nil
            case <-ticker.C:
                fmt.Println("工作中...")
            }
        }
    })

    // 启动并等待退出（Ctrl+C）
    if err := manager.Run(); err != nil {
        fmt.Printf("错误: %v\n", err)
    }
}
```

## 核心功能

### 1. 协程管理

添加和管理应用内的所有协程。

```go
manager := lifecycle.NewManager()

// 添加协程
manager.AddWorker("task1", func(ctx context.Context) error {
    // 协程逻辑
    <-ctx.Done()
    return nil
})

// 添加带停止函数的协程
manager.AddWorker("task2",
    func(ctx context.Context) error {
        // 运行逻辑
        return server.ListenAndServe()
    },
    lifecycle.WithStopFunc(func(ctx context.Context) error {
        // 停止逻辑
        return server.Shutdown(ctx)
    }),
)

// 移除协程
manager.RemoveWorker("task1")
```

### 2. 信号监听

自动监听系统信号，支持自定义信号。

```go
manager := lifecycle.NewManager(
    // 自定义监听的信号
    lifecycle.WithSignals(syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP),
)

// 手动触发退出
go func() {
    time.Sleep(10 * time.Second)
    manager.Shutdown()
}()

manager.Run()
```

### 3. 钩子系统

在生命周期的关键节点执行自定义逻辑。

```go
manager := lifecycle.NewManager()

// 应用启动时
manager.OnStartup(func(ctx context.Context) error {
    fmt.Println("初始化数据库连接...")
    return db.Connect()
})

// 协程启动时
manager.OnWorkerStart(func(name string, err error) {
    fmt.Printf("协程 %s 已启动\n", name)
})

// 协程退出时
manager.OnWorkerExit(func(name string, err error) {
    if err != nil {
        fmt.Printf("协程 %s 异常退出: %v\n", name, err)
    } else {
        fmt.Printf("协程 %s 正常退出\n", name)
    }
})

// 应用退出时
manager.OnShutdown(func(ctx context.Context) error {
    fmt.Println("关闭数据库连接...")
    return db.Close()
})

// 退出超时时
manager.OnTimeout(func(ctx context.Context) error {
    fmt.Println("退出超时，强制关闭")
    return nil
})
```

### 4. 超时控制

设置退出超时时间，防止协程阻塞。

```go
manager := lifecycle.NewManager(
    // 设置30秒退出超时
    lifecycle.WithShutdownTimeout(30 * time.Second),
)

manager.AddWorker("worker", func(ctx context.Context) error {
    <-ctx.Done()
    // 清理工作
    time.Sleep(5 * time.Second)
    return nil
})

manager.Run()
```

### 5. Context 管理

自动传播 Context，支持自定义根 Context。

```go
// 使用自定义根 Context
rootCtx := context.WithValue(context.Background(), "key", "value")

manager := lifecycle.NewManager(
    lifecycle.WithContext(rootCtx),
)

manager.AddWorker("worker", func(ctx context.Context) error {
    // ctx 继承自 rootCtx
    value := ctx.Value("key")
    fmt.Println(value) // 输出: value
    return nil
})
```

## 退出机制

### 两种退出方式

协程可以通过两种方式退出:

1. **Context 取消** - 适用于普通协程
   ```go
   manager.AddWorker("worker", func(ctx context.Context) error {
       for {
           select {
           case <-ctx.Done():
               // Context 被取消，执行清理并退出
               return nil
           case work := <-workChan:
               process(work)
           }
       }
   })
   ```

2. **StopFunc** - 适用于需要主动停止的服务（如 HTTP 服务器）
   ```go
   manager.AddWorker("http-server",
       func(ctx context.Context) error {
           // 阻塞式运行
           return server.ListenAndServe()
       },
       lifecycle.WithStopFunc(func(ctx context.Context) error {
           // 主动停止服务
           return server.Shutdown(ctx)
       }),
   )
   ```

### 退出流程

1. 收到退出信号或手动调用 `Shutdown()`
2. 取消根 Context，通知所有协程
3. **按 LIFO 顺序调用所有 StopFunc**（主动停止服务）
4. 等待所有协程退出（带超时）
5. 调用 OnShutdown 钩子
6. 返回错误（如果有）

**重要**: StopFunc 会在等待协程退出之前调用，确保阻塞式服务能够立即开始关闭流程。

## 应用场景

### 场景 1: HTTP 服务器优雅退出

```go
manager := lifecycle.NewManager(
    lifecycle.WithShutdownTimeout(5 * time.Second),
)

server := &http.Server{Addr: ":8080"}

manager.AddWorker("http-server",
    func(ctx context.Context) error {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            return err
        }
        return nil
    },
    lifecycle.WithStopFunc(func(ctx context.Context) error {
        return server.Shutdown(ctx)
    }),
)

manager.Run()
```

### 场景 2: 后台任务管理

```go
manager := lifecycle.NewManager()

// 定时任务
manager.AddWorker("cron", func(ctx context.Context) error {
    ticker := time.NewTicker(time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            cleanupExpiredData()
        }
    }
})

// 消息队列消费者
manager.AddWorker("consumer", func(ctx context.Context) error {
    return consumeMessages(ctx)
})

manager.Run()
```

### 场景 3: 微服务应用

```go
manager := lifecycle.NewManager(
    lifecycle.WithShutdownTimeout(30 * time.Second),
)

// 初始化资源
manager.OnStartup(func(ctx context.Context) error {
    db.Connect()
    cache.Connect()
    return nil
})

// HTTP服务
manager.AddWorker("http", httpServerWorker)

// gRPC服务
manager.AddWorker("grpc", grpcServerWorker)

// 健康检查
manager.AddWorker("health", healthCheckWorker)

// 清理资源
manager.OnShutdown(func(ctx context.Context) error {
    db.Close()
    cache.Close()
    return nil
})

manager.Run()
```

### 场景 4: 长连接服务

```go
manager := lifecycle.NewManager()

manager.AddWorker("websocket", func(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            // 关闭所有连接
            closeAllConnections()
            return nil
        case conn := <-newConnChan:
            go handleConnection(ctx, conn)
        }
    }
})

manager.Run()
```

## API 参考

### Manager 方法

| 方法 | 说明 |
|------|------|
| `NewManager(opts ...Option)` | 创建生命周期管理器 |
| `AddWorker(name, runFunc, opts...)` | 添加协程 |
| `RemoveWorker(name)` | 移除协程 |
| `OnStartup(fn)` | 注册启动钩子 |
| `OnWorkerStart(fn)` | 注册协程启动钩子 |
| `OnWorkerExit(fn)` | 注册协程退出钩子 |
| `OnShutdown(fn)` | 注册退出钩子 |
| `OnTimeout(fn)` | 注册超时钩子 |
| `Run()` | 启动管理器并等待退出 |
| `Shutdown()` | 手动触发退出 |

### 配置选项

| 选项 | 说明 |
|------|------|
| `WithSignals(signals...)` | 设置监听的信号 |
| `WithShutdownTimeout(timeout)` | 设置退出超时时间 |
| `WithContext(ctx)` | 设置根 Context |

### Worker 选项

| 选项 | 说明 |
|------|------|
| `WithStopFunc(fn)` | 设置停止函数 |

## 最佳实践

1. **合理设置超时时间**
   ```go
   manager := lifecycle.NewManager(
       lifecycle.WithShutdownTimeout(30 * time.Second),
   )
   ```

2. **使用 StopFunc 清理资源**
   ```go
   manager.AddWorker("server", runFunc,
       lifecycle.WithStopFunc(func(ctx context.Context) error {
           return server.Shutdown(ctx)
       }),
   )
   ```

3. **监听 Context 取消信号**
   ```go
   manager.AddWorker("worker", func(ctx context.Context) error {
       for {
           select {
           case <-ctx.Done():
               return nil // 正常退出
           case work := <-workChan:
               process(work)
           }
       }
   })
   ```

4. **使用钩子进行资源管理**
   ```go
   manager.OnStartup(func(ctx context.Context) error {
       return initResources()
   })

   manager.OnShutdown(func(ctx context.Context) error {
       return cleanupResources()
   })
   ```

5. **错误处理**
   ```go
   manager.OnWorkerExit(func(name string, err error) {
       if err != nil {
           log.Printf("Worker %s failed: %v", name, err)
           // 记录错误、发送告警等
       }
   })
   ```

## 许可证

MIT License
