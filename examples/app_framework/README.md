# Go 应用程序开发框架模板

基于 go-kitbox 工具库构建的生产级 Go 应用程序开发框架模板，遵循最佳实践。

## 特性

- **配置管理** - 使用 config 包，支持 YAML 配置和热重载
- **日志系统** - 使用 logger 包，结构化日志和日志轮转
- **生命周期管理** - 使用 lifecycle 包，优雅退出和协程管理
- **分层架构** - 清晰的代码组织结构
- **健康检查** - 内置健康检查端点
- **信号处理** - 自动处理 SIGINT/SIGTERM 信号

## 项目结构

```
app_framework/
├── cmd/
│   └── server/
│       └── main.go           # 应用入口
├── internal/
│   ├── config/
│   │   └── config.go         # 配置结构定义
│   ├── handler/
│   │   ├── health.go         # 健康检查处理器
│   │   └── api.go            # API处理器
│   └── service/
│       └── demo.go           # 业务服务
├── configs/
│   └── config.yaml           # 配置文件
└── README.md
```

## 快速开始

### 1. 运行应用

```bash
cd examples/app_framework
go run cmd/server/main.go
```

### 2. 测试接口

```bash
# 健康检查
curl http://localhost:8080/health

# API接口
curl http://localhost:8080/api/demo
```

### 3. 优雅退出

按 `Ctrl+C` 或发送 SIGTERM 信号，应用会优雅退出。

## 配置说明

配置文件位于 `configs/config.yaml`：

```yaml
server:
  name: "demo-server"          # 服务名称
  version: "1.0.0"             # 版本号
  http:
    addr: ":8080"              # HTTP监听地址
    read_timeout: 30s          # 读超时
    write_timeout: 30s         # 写超时
    idle_timeout: 60s          # 空闲超时
  shutdown_timeout: 30s        # 退出超时

logger:
  level: "info"                # 日志级别
  encoding: "json"             # 日志格式
  enable_caller: true          # 启用调用者信息

business:
  cache_ttl: 300s              # 缓存TTL
  max_workers: 10              # 最大工作协程数
```

## 核心组件

### 1. 配置管理

使用 `config` 包加载和管理配置：

```go
cfg := &config.Config{}
mgr := pkgConfig.NewConfigManager(cfg)
mgr.LoadConfig("configs/config.yaml")
```

### 2. 日志系统

使用 `logger` 包记录结构化日志：

```go
log := logger.New(os.Stdout, logger.InfoLevel, zap.AddCaller())
log.Info("消息", logger.String("key", "value"))
```

### 3. 生命周期管理

使用 `lifecycle` 包管理应用生命周期：

```go
lm := lifecycle.NewManager(
    lifecycle.WithShutdownTimeout(cfg.Server.ShutdownTimeout),
)

// 添加HTTP服务器
lm.AddWorker("http-server", runFunc, lifecycle.WithStopFunc(stopFunc))

// 注册钩子
lm.OnStartup(startupFunc)
lm.OnShutdown(shutdownFunc)

// 启动应用
lm.Run()
```

## 扩展指南

### 添加新的API端点

1. 在 `internal/handler` 创建新的处理器
2. 在 `cmd/server/main.go` 注册路由
3. 在 `internal/service` 实现业务逻辑

### 添加数据库支持

```go
// 1. 在配置中添加数据库配置
type Config struct {
    Database DatabaseConfig `yaml:"database"`
}

// 2. 在启动钩子中初始化数据库
lm.OnStartup(func(ctx context.Context) error {
    return db.Connect(cfg.Database)
})

// 3. 在退出钩子中关闭数据库
lm.OnShutdown(func(ctx context.Context) error {
    return db.Close()
})
```

### 添加后台任务

```go
lm.AddWorker("background-task", func(ctx context.Context) error {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil
        case <-ticker.C:
            // 执行定时任务
        }
    }
})
```

### 添加中间件

```go
// 日志中间件
func LoggingMiddleware(next http.Handler, log *logger.Logger) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Info("请求处理",
            logger.String("method", r.Method),
            logger.String("path", r.URL.Path),
            logger.Duration("duration", time.Since(start)),
        )
    })
}

// 使用中间件
mux.Handle("/api/demo", LoggingMiddleware(apiHandler, log))
```

## 最佳实践

### 1. 错误处理

```go
if err != nil {
    log.Error("操作失败",
        logger.Error(err),
        logger.String("operation", "xxx"),
    )
    return err
}
```

### 2. Context 传递

```go
func (s *Service) Process(ctx context.Context) error {
    // 使用 context 进行超时控制和取消
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // 传递给下游
    return s.repo.Query(ctx)
}
```

### 3. 资源清理

```go
lm.OnShutdown(func(ctx context.Context) error {
    // 按顺序清理资源
    db.Close()
    cache.Close()
    return nil
})
```

### 4. 配置热重载

```go
mgr.Watch(func(cfg interface{}) {
    log.Info("配置已更新")
    // 重新加载配置
})
```

## 构建和部署

### 1. 构建二进制文件

```bash
# 使用 Makefile
make build

# 或直接使用 go build
go build -o bin/server cmd/server/main.go
```

### 2. 运行

```bash
# 直接运行
./bin/server

# 或使用 Makefile
make run
```

### 3. 环境变量

```bash
# 指定配置文件路径
export CONFIG_PATH=/path/to/config.yaml
./bin/server
```

## 监控和可观测性

### 健康检查

```bash
curl http://localhost:8080/health
```

### 日志

应用使用结构化日志，可以轻松集成到日志收集系统（如 ELK、Loki）。

### 指标

可以集成 Prometheus 指标：

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

mux.Handle("/metrics", promhttp.Handler())
```

## 许可证

MIT License
