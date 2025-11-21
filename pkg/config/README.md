# Config - 配置管理

通用配置管理器，支持多种格式和热重载。

## 特性

- 支持 YAML/JSON 格式，可扩展其他格式
- 支持无后缀配置文件（通过强制指定格式）
- 配置监听实现热更新，无需重启服务
- 并发安全，可在多 goroutine 中安全使用
- 灵活的路径配置和格式适配

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/config
go get github.com/fsnotify/fsnotify
go get gopkg.in/yaml.v2
```

## 快速开始

### 1. 定义配置结构体

```go
type AppConfig struct {
    Server struct {
        Port    int    `yaml:"port" json:"port"`
        Host    string `yaml:"host" json:"host"`
        Timeout int    `yaml:"timeout" json:"timeout"`
    } `yaml:"server" json:"server"`
    Logger struct {
        Level  string `yaml:"level" json:"level"`
        Path   string `yaml:"path" json:"path"`
        Rotate bool   `yaml:"rotate" json:"rotate"`
    } `yaml:"logger" json:"logger"`
    Database struct {
        DSN string `yaml:"dsn" json:"dsn"`
    } `yaml:"database" json:"database"`
}
```

### 2. 创建配置文件

创建 `app.yml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  timeout: 30
logger:
  level: "info"
  path: "./logs/app.log"
  rotate: true
database:
  dsn: "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8"
```

### 3. 加载配置

```go
package main

import (
    "log"
    "github.com/junbin-yang/go-kitbox/pkg/config"
)

func main() {
    // 创建配置实例（必须传指针）
    cfg := &AppConfig{}

    // 创建配置管理器
    cm := config.NewConfigManager(
        cfg,
        config.WithAppName("app"), // 对应配置文件名 app.yml
    )

    // 加载配置（空字符串表示使用默认路径）
    if err := cm.LoadConfig(""); err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }

    // 获取配置并使用
    configData, err := cm.GetConfig()
    if err != nil {
        log.Fatalf("获取配置失败: %v", err)
    }

    appConfig := configData.(*AppConfig)
    log.Printf("服务器端口: %d", appConfig.Server.Port)
    log.Printf("日志级别: %s", appConfig.Logger.Level)
}
```

## 使用场景

### 场景 1：基础使用（默认 YAML 格式）

```go
cfg := &AppConfig{}
cm := config.NewConfigManager(cfg, config.WithAppName("myapp"))
cm.LoadConfig("") // 自动查找 myapp.yml 或 myapp
```

### 场景 2：使用 JSON 格式配置

创建 `app.json`:

```json
{
  "server": {
    "port": 8080,
    "host": "0.0.0.0"
  },
  "logger": {
    "level": "debug"
  }
}
```

```go
cm := config.NewConfigManager(
    cfg,
    config.WithAppName("app"),
    config.WithSerializer(&config.JSONSerializer{}), // 指定 JSON 格式
)
cm.LoadConfig("") // 自动查找 app.json
```

### 场景 3：加载无后缀配置文件

```go
cm := config.NewConfigManager(
    cfg,
    config.WithForceFormat(&config.YAMLSerializer{}), // 强制按 YAML 解析
    config.WithDefaultPaths("./config"),
)
cm.LoadConfig("") // 加载 ./config 并按 YAML 解析
```

### 场景 4：加载自定义路径的配置

```go
cm := config.NewConfigManager(
    cfg,
    config.WithForceFormat(&config.JSONSerializer{}),
)
cm.LoadConfig("./myconfig") // 加载 ./myconfig 并按 JSON 解析
```

### 场景 5：启用配置监听（热重载）

```go
cm := config.NewConfigManager(
    cfg,
    config.WithAppName("app"),
    config.WithConfigWatch(true, 500*time.Millisecond), // 启用监听，防抖 500ms
)
cm.LoadConfig("")

// 文件变化时自动重载
// 控制台会输出：[CONFIG] auto reloaded from: ./app.yml
```

### 场景 6：修改配置并保存

```go
conf, _ := cm.GetConfig()
appConf := conf.(*AppConfig)

// 修改配置
appConf.Logger.Level = "debug"
appConf.Server.Port = 9090

// 保存到文件
if err := cm.SaveConfig(); err != nil {
    log.Fatalf("保存配置失败: %v", err)
}
```

### 场景 7：手动重载配置

```go
// 外部修改配置文件后，手动触发重载
if err := cm.ReloadConfig(); err != nil {
    log.Printf("重载配置失败: %v", err)
} else {
    log.Println("配置重载成功")
}
```

### 场景 8：动态控制监听

```go
cm.EnableWatch(false) // 停止监听
cm.EnableWatch(true)  // 启动监听
```

### 场景 9：自定义配置路径

```go
cm := config.NewConfigManager(
    cfg,
    config.WithAppName("myapp"),
    config.WithDefaultPaths(
        "./conf/{{.AppName}}.yml",      // 先查 ./conf/myapp.yml
        "~/.{{.AppName}}/config.yml",   // 再查用户目录
        "/etc/{{.AppName}}/config.yml", // 最后查系统目录
    ),
)
cm.LoadConfig("")
```

### 场景 10：关闭配置管理器

```go
defer cm.Close() // 停止监听并释放资源
```

## API 参考

### 配置选项

| 选项 | 说明 | 示例 |
|------|------|------|
| `WithAppName(name)` | 设置应用名称（用于默认配置文件名） | `WithAppName("myapp")` |
| `WithSerializer(s)` | 设置默认序列化器 | `WithSerializer(&JSONSerializer{})` |
| `WithForceFormat(s)` | 强制指定配置格式（无视文件后缀） | `WithForceFormat(&YAMLSerializer{})` |
| `WithDefaultPaths(paths...)` | 设置默认配置文件查找路径 | `WithDefaultPaths("./config")` |
| `WithConfigFormats(formats...)` | 设置支持的配置格式列表 | `WithConfigFormats(&JSONSerializer{})` |
| `WithConfigWatch(enable, interval)` | 启用配置文件监听 | `WithConfigWatch(true, 1*time.Second)` |

### 核心方法

| 方法 | 说明 | 使用场景 |
|------|------|----------|
| `NewConfigManager(cfg, options...)` | 创建配置管理器实例 | 初始化配置 |
| `LoadConfig(customPath)` | 加载配置文件 | 启动时加载配置 |
| `GetConfig()` | 获取配置实例 | 业务中读取配置 |
| `SaveConfig()` | 将内存配置保存到文件 | 修改配置后持久化 |
| `ReloadConfig()` | 手动重载配置 | 外部修改配置文件后更新 |
| `EnableWatch(enable)` | 动态启用/禁用配置监听 | 运行时控制监听开关 |
| `Close()` | 关闭配置管理器（停止监听） | 程序退出时释放资源 |

## 最佳实践

1. **配置结构体定义**：同时添加 `yaml` 和 `json` 标签，保证格式兼容性
2. **指针传递**：配置实例必须传指针给 `NewConfigManager`
3. **错误处理**：始终检查 `LoadConfig` 和 `GetConfig` 的错误返回值
4. **资源释放**：使用 `defer cm.Close()` 确保资源正确释放
5. **热重载场景**：Web 服务适合启用配置监听，工具类程序可直接加载配置使用
6. **并发安全**：配置管理器内部使用读写锁，可在多 goroutine 中安全使用

## 示例代码

完整示例请参考 [examples/config_example](../../examples/config_example/)

## 许可证

MIT License
