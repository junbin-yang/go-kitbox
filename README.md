# go-kitbox

[![CI](https://github.com/junbin-yang/go-kitbox/actions/workflows/ci.yml/badge.svg)](https://github.com/junbin-yang/go-kitbox/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/junbin-yang/go-kitbox/branch/master/graph/badge.svg)](https://codecov.io/gh/junbin-yang/go-kitbox)
[![Go Report Card](https://goreportcard.com/badge/github.com/junbin-yang/go-kitbox)](https://goreportcard.com/report/github.com/junbin-yang/go-kitbox)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go 语言工具库集合，遵循最佳实践，提供一系列实用工具包以简化 Go 应用开发。

## 安装

```bash
go get github.com/junbin-yang/go-kitbox
```

## 工具包

| 包名                              | 功能描述                                             | 文档                                  | 示例                                      |
| --------------------------------- | ---------------------------------------------------- | ------------------------------------- | ----------------------------------------- |
| [bytesconv](pkg/bytesconv/)       | 高性能的零拷贝字符串与字节切片转换                   | [📖 文档](pkg/bytesconv/README.md)    | [💡 示例](examples/bytesconv_example/)    |
| [config](pkg/config/)             | 通用配置管理器，支持多种格式、热重载、环境变量注入等 | [📖 文档](pkg/config/README.md)       | [💡 示例](examples/config_example/)       |
| [logger](pkg/logger/)             | 统一的日志接口，默认基于 zap 实现，支持自定义日志库  | [📖 文档](pkg/logger/README.md)       | [💡 示例](examples/logger_example/)       |
| [timer](pkg/timer/)               | 定时器管理，支持防抖、节流、重试等功能               | [📖 文档](pkg/timer/README.md)        | [💡 示例](examples/timer_example/)        |
| [statemachine](pkg/statemachine/) | 状态机工具库，支持 FSM/HSM/并发/异步状态机           | [📖 文档](pkg/statemachine/README.md) | [💡 示例](examples/statemachine_example/) |
| [lifecycle](pkg/lifecycle/)       | 应用生命周期管理，支持优雅退出和协程管理             | [📖 文档](pkg/lifecycle/README.md)    | [💡 示例](examples/lifecycle_example/)    |
| [taskpool](pkg/taskpool/)         | 高性能任务协程池，支持优先级队列和动态扩缩容         | [📖 文档](pkg/taskpool/README.md)     | [💡 示例](examples/taskpool_example/)     |
| [netconn](pkg/netconn/)           | 统一网络连接库，支持 TCP 和 UDP（FILLP）             | [📖 文档](pkg/netconn/README.md)      | [💡 示例](examples/netconn_example/)      |
| [fillp](pkg/fillp/)               | 基于 UDP 的可靠传输协议（类 TCP）                    | [📖 文档](pkg/fillp/README.md)        | [💡 示例](examples/fillp_example/)        |
| [congestion](pkg/congestion/)     | 网络拥塞控制算法（CUBIC/BBR/Reno/Vegas）             | [📖 文档](pkg/congestion/README.md)   | [💡 示例](examples/fillp_example/)        |
| [binpack](pkg/binpack/)           | 二进制协议编解码器，支持代码生成和零反射开销         | [📖 文档](pkg/binpack/README.md)      | [💡 示例](examples/binpack_example/)      |

## 测试

```bash
make test
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE)
