# Logger - 日志库

基于 [zap](https://github.com/uber-go/zap) 的二次封装，提供更简洁的 API 和开箱即用的日志轮转功能。

## 特性

- 基于高性能的 zap 封装
- 开箱即用的默认 logger
- 支持日志轮转（按时间/按大小）
- 动态调整日志级别
- 完整的 zap 选项支持

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/logger
go get go.uber.org/zap
go get github.com/lestrrat-go/file-rotatelogs
go get gopkg.in/natefinch/lumberjack.v2
```

## 快速开始

### 基础使用

```go
package main

import (
    "time"
    log "github.com/junbin-yang/go-kitbox/pkg/logger"
)

func main() {
    defer log.Sync()
    
    log.Info("Info msg")
    log.Warn("Warn msg", log.Int("attempt", 3))
    log.Error("Error msg", log.Duration("backoff", time.Second))
}
```

### 动态调整日志级别

```go
// 修改日志级别
log.SetLevel(log.ErrorLevel)
log.Info("不会输出")
log.Error("会输出")
```

### 自定义 Logger

```go
file, _ := os.OpenFile("custom.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
logger := log.New(file, log.InfoLevel)
log.ReplaceDefault(logger)
```

## 日志轮转

### 按时间轮转

适用于需要按固定时间间隔归档日志的场景。

```go
package main

import (
    "time"
    log "github.com/junbin-yang/go-kitbox/pkg/logger"
)

func main() {
    // 使用默认配置（24小时轮转，保留30天）
    out := log.NewProductionRotateByTime("app.log")
    logger := log.New(out, log.InfoLevel)
    log.ReplaceDefault(logger)
    defer log.Sync()

    log.Info("日志会按时间轮转")
}
```

**默认配置：**
- 轮转间隔：24 小时
- 保留时间：30 天
- 时间格式：UTC
- 文件命名：`app.2024-01-15-10-30-00.log`

### 按大小轮转

适用于日志量大、需要控制单个文件大小的场景。

```go
package main

import (
    log "github.com/junbin-yang/go-kitbox/pkg/logger"
)

func main() {
    // 使用默认配置（100MB轮转，保留100个文件）
    out := log.NewProductionRotateBySize("app.log")
    logger := log.New(out, log.InfoLevel)
    log.ReplaceDefault(logger)
    defer log.Sync()

    log.Info("日志会按大小轮转")
}
```

**默认配置：**
- 单文件大小：100 MB
- 保留文件数：100 个
- 保留时间：30 天
- 自动压缩：是

### 自定义轮转配置

```go
package main

import (
    "time"
    log "github.com/junbin-yang/go-kitbox/pkg/logger"
)

func main() {
    // 自定义按时间轮转
    cfg := &log.RotateConfig{
        Filename:     "custom.log",
        MaxAge:       7,              // 保留7天
        RotationTime: time.Hour * 1,  // 每小时轮转
        LocalTime:    true,           // 使用本地时间
    }
    out := log.NewRotateByTime(cfg)
    logger := log.New(out, log.InfoLevel)
    
    // 自定义按大小轮转
    cfg2 := &log.RotateConfig{
        Filename:   "size.log",
        MaxSize:    50,    // 50MB
        MaxBackups: 10,    // 保留10个文件
        MaxAge:     7,     // 保留7天
        Compress:   true,  // 压缩旧文件
        LocalTime:  true,  // 使用本地时间
    }
    out2 := log.NewRotateBySize(cfg2)
    logger2 := log.New(out2, log.InfoLevel)
    
    logger.Info("按时间轮转")
    logger2.Info("按大小轮转")
}
```

### 轮转配置说明

| 配置项 | 类型 | 说明 | 默认值 |
|--------|------|------|--------|
| `Filename` | string | 日志文件路径 | - |
| `MaxAge` | int | 保留天数 | 30 |
| `RotationTime` | Duration | 轮转间隔（按时间） | 24h |
| `MaxSize` | int | 单文件大小 MB（按大小） | 100 |
| `MaxBackups` | int | 保留文件数（按大小） | 100 |
| `Compress` | bool | 是否压缩（按大小） | true |
| `LocalTime` | bool | 使用本地时间 | false |

## 高级用法

### 使用 Zap 选项

```go
package main

import (
    "fmt"
    "io"
    "os"
    
    "go.uber.org/zap/zapcore"
    log "github.com/junbin-yang/go-kitbox/pkg/logger"
)

func main() {
    file, _ := os.OpenFile("test.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    opts := []log.Option{
        log.WithCaller(true),      // 显示调用位置
        log.AddCallerSkip(1),      // 跳过调用栈
        log.Hooks(func(entry zapcore.Entry) error {
            if entry.Level == log.WarnLevel {
                fmt.Printf("Warn Hook: %s\n", entry.Message)
            }
            return nil
        }),
    }
    logger := log.New(io.MultiWriter(os.Stdout, file), log.InfoLevel, opts...)
    defer logger.Sync()

    logger.Info("Info msg", log.String("key", "value"))
    logger.Warn("Warn msg", log.Int("code", 500))
}
```

### 结构化日志字段

```go
log.Info("user login",
    log.String("username", "alice"),
    log.Int("user_id", 123),
    log.Duration("latency", time.Millisecond*50),
    log.Bool("success", true),
)
```

### 格式化日志

```go
log.Infof("User %s logged in with ID %d", "alice", 123)
log.Errorf("Failed to connect: %v", err)
```

## API 参考

### 日志级别

- `DebugLevel` - 调试信息
- `InfoLevel` - 一般信息
- `WarnLevel` - 警告信息
- `ErrorLevel` - 错误信息
- `PanicLevel` - Panic 级别
- `FatalLevel` - Fatal 级别

### 核心方法

| 方法 | 说明 |
|------|------|
| `New(out, level, opts...)` | 创建新 logger |
| `Default()` | 获取默认 logger |
| `ReplaceDefault(logger)` | 替换默认 logger |
| `SetLevel(level)` | 设置日志级别 |
| `Debug/Info/Warn/Error/Panic/Fatal(msg, fields...)` | 结构化日志 |
| `Debugf/Infof/Warnf/Errorf/Panicf/Fatalf(format, args...)` | 格式化日志 |
| `Sync()` | 刷新缓冲区 |

### 字段类型

支持所有 zap 字段类型：`String`, `Int`, `Bool`, `Duration`, `Time`, `Error`, `Any` 等。

## 最佳实践

1. **程序退出前调用 Sync**：确保日志完全写入
   ```go
   defer log.Sync()
   ```

2. **使用结构化日志**：便于日志分析
   ```go
   log.Info("operation", log.String("op", "create"), log.Int("id", 123))
   ```

3. **生产环境使用日志轮转**：避免日志文件过大
   ```go
   out := log.NewProductionRotateBySize("app.log")
   ```

4. **合理设置日志级别**：开发用 Debug，生产用 Info

## 许可证

MIT License
