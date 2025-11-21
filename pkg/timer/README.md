# Timer - 定时器管理库

提供定时器管理功能，包括周期性定时器、一次性定时器及防抖、节流、重试等实用工具函数。

## 特性

-   周期性定时器和一次性定时器
-   定时器动态管理（创建、停止、重置）
-   防抖（Debounce）和节流（Throttle）
-   重试机制（固定间隔和指数退避）
-   并发安全
-   Panic 恢复保护

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/timer
```

## 快速开始

### 周期性定时器

```go
package main

import (
    "fmt"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/timer"
)

func main() {
    mgr := timer.NewManager()
    defer mgr.StopAll()

    // 每秒执行一次
    mgr.CreateTimer("task1", time.Second, func() {
        fmt.Println("执行周期性任务")
    })

    time.Sleep(5 * time.Second)
}
```

### 一次性定时器

```go
// 3秒后执行一次
mgr.CreateOnceTimer("once", 3*time.Second, func() {
    fmt.Println("延迟任务执行")
})
```

## 核心功能

### 1. 定时器管理

#### 创建周期性定时器

```go
mgr := timer.NewManager()

// 每2秒执行一次
err := mgr.CreateTimer("heartbeat", 2*time.Second, func() {
    fmt.Println("心跳检测")
})
```

#### 创建一次性定时器

```go
// 5秒后执行
err := mgr.CreateOnceTimer("delayed", 5*time.Second, func() {
    fmt.Println("延迟任务")
})
```

#### 停止定时器

```go
// 停止指定定时器
err := mgr.RemoveTimer("heartbeat")

// 停止所有定时器
mgr.StopAll()
```

#### 重置定时器间隔

```go
// 周期性定时器：改为5秒间隔
err := mgr.ResetTimer("heartbeat", 5*time.Second)

// 一次性定时器：重新设置延迟时间（未执行或已执行都可重置）
err := mgr.ResetTimer("delayed", 3*time.Second)
```

#### 查询定时器

```go
// 获取定时器信息
info, exists := mgr.GetTimer("heartbeat")
if exists {
    fmt.Printf("间隔: %v, 一次性: %v\n", info.Interval, info.IsOnce)
}

// 获取所有定时器ID
ids := mgr.ListTimers()

// 获取定时器数量
count := mgr.GetTimerCount()
```

### 2. 防抖（Debounce）

多次调用时，仅在最后一次调用后等待指定时间再执行。适用于搜索输入框等场景。

```go
package main

import (
    "fmt"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/timer"
)

func main() {
    // 创建防抖函数，500ms内无新调用才执行
    search := timer.Debounce(500*time.Millisecond, func() {
        fmt.Println("执行搜索")
    })

    // 模拟用户快速输入
    search() // 不会执行
    time.Sleep(100 * time.Millisecond)
    search() // 不会执行
    time.Sleep(100 * time.Millisecond)
    search() // 500ms后执行

    time.Sleep(time.Second)
}
```

### 3. 节流（Throttle）

指定时间内最多执行一次。适用于按钮点击、滚动事件等场景。

```go
package main

import (
    "fmt"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/timer"
)

func main() {
    // 创建节流函数，1秒内最多执行一次
    onClick := timer.Throttle(time.Second, func() {
        fmt.Println("按钮点击")
    })

    // 快速点击
    onClick() // 执行
    onClick() // 忽略
    onClick() // 忽略
    time.Sleep(1100 * time.Millisecond)
    onClick() // 执行
}
```

### 4. 重试机制

#### 固定间隔重试

```go
package main

import (
    "errors"
    "fmt"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/timer"
)

func main() {
    attempt := 0
    err := timer.Retry(3, time.Second, func() error {
        attempt++
        fmt.Printf("尝试 %d\n", attempt)
        if attempt < 3 {
            return errors.New("失败")
        }
        return nil
    })

    if err != nil {
        fmt.Println("最终失败:", err)
    } else {
        fmt.Println("成功")
    }
}
```

#### 指数退避重试

```go
package main

import (
    "errors"
    "fmt"
    "time"
    "github.com/junbin-yang/go-kitbox/pkg/timer"
)

func main() {
    attempt := 0
    err := timer.ExponentialBackoff(4, 100*time.Millisecond, func() error {
        attempt++
        fmt.Printf("尝试 %d\n", attempt)
        if attempt < 3 {
            return errors.New("网络错误")
        }
        return nil
    })
    // 重试间隔: 100ms, 200ms, 400ms
}
```

### 5. 字符串解析定时器

```go
// 支持 "5s", "1m", "2h" 等格式
err := mgr.ScheduleFunc("task", "30s", func() {
    fmt.Println("每30秒执行")
})
```

## 应用场景

### 场景 1：心跳检测

```go
mgr := timer.NewManager()
defer mgr.StopAll()

mgr.CreateTimer("heartbeat", 10*time.Second, func() {
    // 发送心跳包
    sendHeartbeat()
})
```

### 场景 2：缓存过期清理

```go
mgr.CreateTimer("cache-cleanup", time.Hour, func() {
    cleanExpiredCache()
})
```

### 场景 3：搜索防抖

```go
searchDebounced := timer.Debounce(300*time.Millisecond, func() {
    performSearch(query)
})

// 用户每次输入都调用
onInputChange := func(q string) {
    query = q
    searchDebounced()
}
```

### 场景 4：API 请求节流

```go
apiCall := timer.Throttle(time.Second, func() {
    callAPI()
})

// 高频调用，但实际1秒最多执行1次
for i := 0; i < 100; i++ {
    apiCall()
}
```

### 场景 5：网络请求重试

```go
err := timer.ExponentialBackoff(5, 100*time.Millisecond, func() error {
    return httpClient.Get(url)
})
```

### 场景 6：延迟任务

```go
// 用户注册后5分钟发送欢迎邮件
mgr.CreateOnceTimer("welcome-email", 5*time.Minute, func() {
    sendWelcomeEmail(userID)
})
```

## API 参考

### Manager 方法

| 方法                                   | 说明                 |
| -------------------------------------- | -------------------- |
| `NewManager()`                         | 创建定时器管理器     |
| `CreateTimer(id, interval, callback)`  | 创建周期性定时器     |
| `CreateOnceTimer(id, delay, callback)` | 创建一次性定时器     |
| `RemoveTimer(id)`                      | 停止并移除定时器     |
| `StopAll()`                            | 停止所有定时器       |
| `ResetTimer(id, newInterval)`          | 重置定时器间隔       |
| `GetTimer(id)`                         | 获取定时器信息       |
| `ListTimers()`                         | 获取所有定时器 ID    |
| `GetTimerCount()`                      | 获取定时器数量       |
| `ScheduleFunc(id, spec, callback)`     | 字符串解析创建定时器 |

### 工具函数

| 函数                                             | 说明         |
| ------------------------------------------------ | ------------ |
| `Debounce(wait, callback)`                       | 创建防抖函数 |
| `Throttle(duration, callback)`                   | 创建节流函数 |
| `Retry(attempts, delay, fn)`                     | 固定间隔重试 |
| `ExponentialBackoff(attempts, initialDelay, fn)` | 指数退避重试 |

## 最佳实践

1. **及时清理定时器**：使用 `defer mgr.StopAll()` 确保程序退出时清理资源

    ```go
    mgr := timer.NewManager()
    defer mgr.StopAll()
    ```

2. **避免回调阻塞**：回调函数应快速返回，耗时操作应启动新 goroutine

    ```go
    mgr.CreateTimer("task", time.Second, func() {
        go heavyWork() // 耗时操作放入新协程
    })
    ```

3. **唯一 ID**：确保定时器 ID 唯一，避免冲突

4. **错误处理**：检查创建和操作定时器的返回错误

    ```go
    if err := mgr.CreateTimer("task", time.Second, callback); err != nil {
        log.Printf("创建定时器失败: %v", err)
    }
    ```

5. **防抖节流选择**：
    - 防抖：用户停止操作后才执行（搜索框）
    - 节流：持续操作时定期执行（滚动加载）

## 许可证

MIT License
