# FILLP 示例

演示 FILLP 协议的客户端-服务端通信。

## 运行示例

### 1. 启动服务端

```bash
go run server.go
```

服务端将监听在 `127.0.0.1:8080`，等待客户端连接。

### 2. 启动客户端

在另一个终端运行：

```bash
go run client.go
```

客户端将连接到服务端，发送3条消息并接收回显。

## 预期输出

**服务端输出：**
```
服务端监听 127.0.0.1:8080
客户端已连接
收到数据: Hello FILLP Server!
收到数据: This is message 2
收到数据: This is message 3
统计: 发送 XXX 字节, 接收 XXX 字节, RTT XXms
```

**客户端输出：**
```
连接服务端...
连接成功
发送数据: Hello FILLP Server!
收到回显: Hello FILLP Server!
发送数据: This is message 2
收到回显: This is message 2
发送数据: This is message 3
收到回显: This is message 3
统计: 发送 XXX 字节, 接收 XXX 字节, RTT XXms
```

## 说明

- 服务端使用 `Listen()` 方法监听指定地址
- 客户端使用 `Connect()` 方法连接到服务端
- 使用 `Send()` 发送数据，`ReceiveWithTimeout()` 接收数据
- FILLP 自动处理数据分片、重传、流量控制等
- 连接关闭时会打印统计信息
