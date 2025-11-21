package main

import (
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/congestion"
	"github.com/junbin-yang/go-kitbox/pkg/fillp"
)

func main() {
	algo := flag.String("algo", "builtin", "拥塞控制算法: builtin, cubic, bbr, vegas, reno")
	flag.Parse()

	// 创建客户端连接
	serverAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 8080,
	}

	var client *fillp.Connection
	var err error

	switch *algo {
	case "cubic":
		client, err = fillp.NewConnectionWithCUBIC(nil, serverAddr)
		fmt.Println("使用 CUBIC 拥塞控制算法")
	case "bbr":
		client, err = fillp.NewConnectionWithBBR(nil, serverAddr)
		fmt.Println("使用 BBR 拥塞控制算法")
	case "vegas":
		client, err = fillp.NewConnectionWithVegas(nil, serverAddr)
		fmt.Println("使用 Vegas 拥塞控制算法")
	case "reno":
		config := fillp.ConnectionConfig{
			CongestionAlgorithm: congestion.AlgorithmReno,
		}
		client, err = fillp.NewConnectionWithConfig(nil, serverAddr, config)
		fmt.Println("使用 Reno 拥塞控制算法")
	default:
		client, err = fillp.NewConnection(nil, serverAddr)
		fmt.Println("使用内置拥塞控制算法")
	}

	if err != nil {
		panic(err)
	}
	defer client.Close()

	// 连接服务端
	fmt.Println("连接服务端...")
	if err := client.Connect(); err != nil {
		panic(err)
	}

	fmt.Println("连接成功")

	// 发送多条消息
	messages := []string{
		"Hello FILLP Server!",
		"This is message 2",
		"This is message 3",
	}

	for _, msg := range messages {
		// 发送数据
		if err := client.Send([]byte(msg)); err != nil {
			panic(err)
		}
		fmt.Printf("发送数据: %s\n", msg)

		// 接收回显
		response, err := client.ReceiveWithTimeout(5 * time.Second)
		if err != nil {
			panic(err)
		}
		fmt.Printf("收到回显: %s\n", string(response))

		time.Sleep(500 * time.Millisecond)
	}

	// 获取统计信息
	stats := client.GetStatistics()
	fmt.Printf("统计: 发送 %d 字节, 接收 %d 字节, RTT %v (%.3fms)\n",
		stats.BytesSent, stats.BytesReceived, stats.RTT, float64(stats.RTT.Microseconds())/1000.0)

	// 打印拥塞控制统计
	if client.UseExternalCC() {
		ccStats := client.GetCongestionStats()
		fmt.Printf("拥塞控制: 窗口 %d, 丢包率 %.2f%%, 发送 %d 包, 丢失 %d 包\n",
			ccStats.CongestionWindow, ccStats.LossRate*100, ccStats.PacketsSent, ccStats.PacketsLost)
	}
}
