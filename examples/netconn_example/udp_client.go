package main

import (
	"fmt"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/netconn"
)

func main() {
	callback := &netconn.BaseListenerCallback{
		OnConnected: func(fd int, connType netconn.ConnectionType, connOpt *netconn.ConnectOption) {
			fmt.Printf("[UDP客户端] 连接成功 fd=%d\n", fd)
		},
		OnDisconnected: func(fd int, connType netconn.ConnectionType) {
			fmt.Printf("[UDP客户端] 连接断开 fd=%d\n", fd)
		},
		OnDataReceived: func(fd int, connType netconn.ConnectionType, buf []byte, used int) int {
			fmt.Printf("[UDP客户端] 收到回显: %s\n", string(buf[:used]))
			return used
		},
	}

	client := netconn.NewBaseClient(nil, callback)
	
	fmt.Println("连接服务器...")
	fd, err := client.ConnectSimple(netconn.ProtocolUDP, "127.0.0.1", 8080)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Printf("连接成功 fd=%d\n", fd)

	// 发送多条消息
	messages := []string{
		"Hello UDP Server!",
		"This is message 2",
		"This is message 3",
	}

	for i, msg := range messages {
		fmt.Printf("发送消息 %d: %s\n", i+1, msg)
		if err := client.SendBytes([]byte(msg)); err != nil {
			fmt.Printf("发送失败: %v\n", err)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("所有消息已发送，等待接收...")
	time.Sleep(2 * time.Second)
}
