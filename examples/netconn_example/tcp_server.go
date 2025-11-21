package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/junbin-yang/go-kitbox/pkg/netconn"
)

func main() {
	server := netconn.NewBaseServer(nil)

	callback := &netconn.BaseListenerCallback{
		OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
			if connType == netconn.ConnectionTypeServer {
				fmt.Printf("[TCP服务器] 监听启动 %s:%d\n", connOpt.LocalSocket.Addr, connOpt.LocalSocket.Port)
			} else {
				fmt.Printf("[TCP服务器] 客户端连接 fd=%d, 地址=%s:%d\n",
					fd, connOpt.RemoteSocket.Addr, connOpt.RemoteSocket.Port)
			}
		},
		OnDisconnected: func(fd int, connType netconn.ConnectionType) {
			if connType == netconn.ConnectionTypeClient {
				fmt.Printf("[TCP服务器] 客户端断开 fd=%d\n", fd)
			}
		},
		OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
			data := string(buf[:used])
			fmt.Printf("[TCP服务器] 收到数据 fd=%d: %s\n", fd, data)
			
			// 回显数据
			response := fmt.Sprintf("服务器收到: %s", data)
			server.SendBytes(fd, []byte(response))
			
			return used
		},
	}

	opt := &netconn.ServerOption{
		Protocol: netconn.ProtocolTCP,
		Addr:     "0.0.0.0",
		Port:     8080,
	}

	if err := server.StartBaseListener(opt, callback); err != nil {
		fmt.Printf("启动失败: %v\n", err)
		return
	}
	defer server.StopBaseListener()

	fmt.Println("等待客户端连接...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\n服务器关闭")
}
