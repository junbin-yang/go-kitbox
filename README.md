# go-kitbox

Go 语言工具库集合，遵循最佳实践。

## 安装

```bash
go get github.com/junbin-yang/go-kitbox
```

## 工具包

| 包名                          | 功能描述                                    | 文档                                | 示例                                   |
| ----------------------------- | ------------------------------------------- | ----------------------------------- | -------------------------------------- |
| [bytesconv](pkg/bytesconv/)   | 高性能的零拷贝字符串与字节切片转换          | [📖 文档](pkg/bytesconv/README.md)  | [💡 示例](examples/bytesconv_example/) |
| [config](pkg/config/)         | 通用配置管理器，支持 YAML/JSON 格式和热重载 | [📖 文档](pkg/config/README.md)     | [💡 示例](examples/config_example/)    |
| [logger](pkg/logger/)         | 基于 zap 封装的日志库，支持日志轮转         | [📖 文档](pkg/logger/README.md)     | [💡 示例](examples/logger_example/)    |
| [timer](pkg/timer/)           | 定时器管理，支持防抖、节流、重试等功能      | [📖 文档](pkg/timer/README.md)      | [💡 示例](examples/timer_example/)     |
| [statemachine](pkg/statemachine/) | 状态机工具库，支持FSM/HSM/并发/异步状态机 | [📖 文档](pkg/statemachine/README.md) | [💡 示例](examples/statemachine_example/) |
| [netconn](pkg/netconn/)       | 统一网络连接库，支持 TCP 和 UDP（FILLP）    | [📖 文档](pkg/netconn/README.md)    | [💡 示例](examples/netconn_example/)   |
| [fillp](pkg/fillp/)           | 基于 UDP 的可靠传输协议（类 TCP）           | [📖 文档](pkg/fillp/README.md)      | [💡 示例](examples/fillp_example/)     |
| [congestion](pkg/congestion/) | 网络拥塞控制算法（CUBIC/BBR/Reno/Vegas）    | [📖 文档](pkg/congestion/README.md) | [💡 示例](examples/fillp_example/)     |

## 测试

```bash
make test
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE)
