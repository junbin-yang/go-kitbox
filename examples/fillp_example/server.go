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

	// 创建服务端连接
	serverAddr := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 8080,
	}

	var server *fillp.Connection
	var err error

	switch *algo {
	case "cubic":
		server, err = fillp.NewConnectionWithCUBIC(serverAddr, nil)
		fmt.Println("使用 CUBIC 拥塞控制算法")
	case "bbr":
		server, err = fillp.NewConnectionWithBBR(serverAddr, nil)
		fmt.Println("使用 BBR 拥塞控制算法")
	case "vegas":
		server, err = fillp.NewConnectionWithVegas(serverAddr, nil)
		fmt.Println("使用 Vegas 拥塞控制算法")
	case "reno":
		config := fillp.ConnectionConfig{
			CongestionAlgorithm: congestion.AlgorithmReno,
		}
		server, err = fillp.NewConnectionWithConfig(serverAddr, nil, config)
		fmt.Println("使用 Reno 拥塞控制算法")
	default:
		server, err = fillp.NewConnection(serverAddr, nil)
		fmt.Println("使用内置拥塞控制算法")
	}

	if err != nil {
		panic(err)
	}
	defer server.Close()

	// 监听连接
	fmt.Println("服务端监听 127.0.0.1:8080")
	if err := server.Listen(); err != nil {
		panic(err)
	}

	fmt.Println("客户端已连接")

	// 接收数据并回显
	for {
		data, err := server.ReceiveWithTimeout(30 * time.Second)
		if err != nil {
			fmt.Println("接收错误:", err)
			break
		}
		fmt.Printf("收到数据: %s\n", string(data))

		// 回显数据
		if err := server.Send(data); err != nil {
			fmt.Println("发送错误:", err)
			break
		}
	}

	// 打印统计信息
	stats := server.GetStatistics()
	fmt.Printf("统计: 发送 %d 字节, 接收 %d 字节, RTT %v (%.3fms)\n",
		stats.BytesSent, stats.BytesReceived, stats.RTT, float64(stats.RTT.Microseconds())/1000.0)

	// 打印拥塞控制统计
	if server.UseExternalCC() {
		ccStats := server.GetCongestionStats()
		fmt.Printf("拥塞控制: 窗口 %d, 丢包率 %.2f%%, 发送 %d 包, 丢失 %d 包\n",
			ccStats.CongestionWindow, ccStats.LossRate*100, ccStats.PacketsSent, ccStats.PacketsLost)
	}
}
