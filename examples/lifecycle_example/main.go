package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/lifecycle"
)

func main() {
	fmt.Println("=== 生命周期管理示例 ===\n")

	// 示例1: HTTP服务器优雅退出
	httpServerExample()

	// 示例2: 多协程管理
	multiWorkerExample()

	// 示例3: 钩子函数
	hooksExample()

	// 示例4: 独立Context和StopWorker
	independentContextExample()

	// 示例5: 动态添加协程
	dynamicWorkerExample()
}

// httpServerExample HTTP服务器优雅退出示例
func httpServerExample() {
	fmt.Println("1. HTTP服务器优雅退出")
	fmt.Println("-------------------")

	manager := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(5 * time.Second),
	)

	// 创建HTTP服务器
	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello, World!")
		}),
	}

	// 添加HTTP服务器协程
	manager.AddWorker("http-server",
		func(ctx context.Context) error {
			fmt.Println("  → HTTP服务器启动在 :8080")
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}
			return nil
		},
		lifecycle.WithStopFunc(func(ctx context.Context) error {
			fmt.Println("  → 正在关闭HTTP服务器...")
			return server.Shutdown(ctx)
		}),
	)

	// 注册退出钩子
	manager.OnShutdown(func(ctx context.Context) error {
		fmt.Println("  → 清理资源完成")
		return nil
	})

	// 模拟运行一段时间后退出
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("\n  → 触发退出...")
		manager.Shutdown()
	}()

	if err := manager.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	fmt.Println()
}

// multiWorkerExample 多协程管理示例
func multiWorkerExample() {
	fmt.Println("2. 多协程管理")
	fmt.Println("-------------------")

	manager := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(3 * time.Second),
	)

	// 添加多个后台任务
	manager.AddWorker("task1", func(ctx context.Context) error {
		fmt.Println("  → Task1 启动")
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("  → Task1 退出")
				return nil
			case <-ticker.C:
				fmt.Println("  → Task1 运行中...")
			}
		}
	})

	manager.AddWorker("task2", func(ctx context.Context) error {
		fmt.Println("  → Task2 启动")
		ticker := time.NewTicker(700 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("  → Task2 退出")
				return nil
			case <-ticker.C:
				fmt.Println("  → Task2 运行中...")
			}
		}
	})

	// 模拟运行后退出
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("\n  → 触发退出...")
		manager.Shutdown()
	}()

	if err := manager.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	fmt.Println()
}

// hooksExample 钩子函数示例
func hooksExample() {
	fmt.Println("3. 钩子函数")
	fmt.Println("-------------------")

	manager := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(2 * time.Second),
	)

	// 注册启动钩子
	manager.OnStartup(func(ctx context.Context) error {
		fmt.Println("  → 应用启动: 初始化资源")
		return nil
	})

	// 注册协程启动钩子
	manager.OnWorkerStart(func(name string, err error) {
		fmt.Printf("  → 协程启动: %s\n", name)
	})

	// 注册协程退出钩子
	manager.OnWorkerExit(func(name string, err error) {
		if err != nil {
			fmt.Printf("  → 协程退出: %s (错误: %v)\n", name, err)
		} else {
			fmt.Printf("  → 协程退出: %s (正常)\n", name)
		}
	})

	// 注册退出钩子
	manager.OnShutdown(func(ctx context.Context) error {
		fmt.Println("  → 应用退出: 清理资源")
		return nil
	})

	// 添加工作协程
	manager.AddWorker("worker", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	// 模拟退出
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Println("\n  → 触发退出...")
		manager.Shutdown()
	}()

	if err := manager.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	fmt.Println("\n完成!")
}

// independentContextExample 独立Context和StopWorker示例
func independentContextExample() {
	fmt.Println("4. 独立Context和StopWorker")
	fmt.Println("-------------------")

	manager := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(3 * time.Second),
	)

	// 添加两个独立的协程
	manager.AddWorker("worker1", func(ctx context.Context) error {
		fmt.Println("  → Worker1 启动")
		<-ctx.Done()
		fmt.Println("  → Worker1 退出")
		return nil
	})

	manager.AddWorker("worker2", func(ctx context.Context) error {
		fmt.Println("  → Worker2 启动")
		<-ctx.Done()
		fmt.Println("  → Worker2 退出")
		return nil
	})

	// 1秒后停止worker1
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Println("\n  → 停止 Worker1...")
		manager.StopWorker("worker1")
	}()

	// 2秒后退出所有
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("\n  → 触发全局退出...")
		manager.Shutdown()
	}()

	if err := manager.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
	}

	fmt.Println()
}

// dynamicWorkerExample 动态添加协程示例
func dynamicWorkerExample() {
	fmt.Println("5. 动态添加协程")
	fmt.Println("-------------------")

	manager := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(3 * time.Second),
	)

	// 添加主协程
	manager.AddWorker("main-worker", func(ctx context.Context) error {
		fmt.Println("  → 主协程启动")
		<-ctx.Done()
		fmt.Println("  → 主协程退出")
		return nil
	})

	// 启动管理器
	go func() {
		if err := manager.Run(); err != nil {
			fmt.Printf("错误: %v\n", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// 运行时动态添加临时任务
	fmt.Println("  → 动态添加临时任务...")
	manager.AddWorker("temp-task", func(ctx context.Context) error {
		fmt.Println("  → 临时任务启动")
		time.Sleep(500 * time.Millisecond)
		fmt.Println("  → 临时任务完成（自动清理）")
		return nil
	})

	// 等待临时任务完成
	time.Sleep(1 * time.Second)

	// 再添加一个临时任务
	fmt.Println("  → 动态添加第二个临时任务...")
	manager.AddWorker("temp-task-2", func(ctx context.Context) error {
		fmt.Println("  → 临时任务2启动")
		time.Sleep(500 * time.Millisecond)
		fmt.Println("  → 临时任务2完成（自动清理）")
		return nil
	})

	time.Sleep(1 * time.Second)

	// 触发退出
	fmt.Println("\n  → 触发退出...")
	manager.Shutdown()

	time.Sleep(500 * time.Millisecond)
	fmt.Println("\n完成!")
}
