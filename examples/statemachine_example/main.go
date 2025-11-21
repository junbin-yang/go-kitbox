package main

import (
	"context"
	"fmt"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/statemachine"
)

func main() {
	fmt.Println("=== 状态机示例 ===\n")

	// 示例1: 基础有限状态机 - 订单状态
	orderExample()

	// 示例2: 层次状态机 - 工作流程
	workflowExample()

	// 示例3: 并发状态机 - 多任务管理
	concurrentExample()

	// 示例4: 异步状态机 - 事件队列
	asyncExample()
}

// orderExample 订单状态机示例
func orderExample() {
	fmt.Println("1. 订单状态机示例")
	fmt.Println("-------------------")

	// 创建状态机
	fsm := statemachine.NewFSM("pending")

	// 定义状态转换
	fsm.AddTransition("pending", "paid", "pay")
	fsm.AddTransition("paid", "shipped", "ship")
	fsm.AddTransition("shipped", "delivered", "deliver")
	fsm.AddTransition("delivered", "completed", "complete")

	// 设置状态回调
	fsm.SetOnEnter("paid", func(ctx context.Context, state statemachine.State) error {
		fmt.Println("  → 订单已支付,准备发货")
		return nil
	})

	fsm.SetOnEnter("shipped", func(ctx context.Context, state statemachine.State) error {
		fmt.Println("  → 订单已发货,物流中")
		return nil
	})

	fsm.SetOnEnter("delivered", func(ctx context.Context, state statemachine.State) error {
		fmt.Println("  → 订单已送达")
		return nil
	})

	ctx := context.Background()

	// 模拟订单流程
	fmt.Printf("当前状态: %s\n", fsm.Current())

	fsm.Trigger(ctx, "pay")
	fmt.Printf("当前状态: %s\n", fsm.Current())

	fsm.Trigger(ctx, "ship")
	fmt.Printf("当前状态: %s\n", fsm.Current())

	fsm.Trigger(ctx, "deliver")
	fmt.Printf("当前状态: %s\n\n", fsm.Current())
}

// workflowExample 层次状态机示例
func workflowExample() {
	fmt.Println("2. 工作流程层次状态机")
	fmt.Println("-------------------")

	hsm := statemachine.NewHSM("idle")

	// 定义层次结构
	hsm.AddState("working", "idle")
	hsm.AddState("coding", "working")
	hsm.AddState("testing", "working")

	// 定义转换
	hsm.AddTransition("idle", "working", "start_work")
	hsm.AddTransition("working", "coding", "start_coding")
	hsm.AddTransition("coding", "testing", "start_testing")
	hsm.AddTransition("working", "idle", "finish_work")

	ctx := context.Background()

	fmt.Printf("当前状态: %s\n", hsm.Current())

	hsm.Trigger(ctx, "start_work")
	fmt.Printf("开始工作 → %s\n", hsm.Current())

	hsm.Trigger(ctx, "start_coding")
	fmt.Printf("开始编码 → %s\n", hsm.Current())

	hsm.Trigger(ctx, "start_testing")
	fmt.Printf("开始测试 → %s\n", hsm.Current())

	// 从子状态触发父状态的事件
	hsm.Trigger(ctx, "finish_work")
	fmt.Printf("完成工作 → %s\n\n", hsm.Current())
}

// concurrentExample 并发状态机示例
func concurrentExample() {
	fmt.Println("3. 并发状态机示例")
	fmt.Println("-------------------")

	concurrent := statemachine.NewConcurrent()

	// 创建认证状态机
	authFSM := statemachine.NewFSM("logged_out")
	authFSM.AddTransition("logged_out", "logged_in", "login")
	authFSM.AddTransition("logged_in", "logged_out", "logout")

	// 创建支付状态机
	paymentFSM := statemachine.NewFSM("unpaid")
	paymentFSM.AddTransition("unpaid", "paid", "pay")
	paymentFSM.AddTransition("paid", "refunded", "refund")

	// 添加到并发管理器
	concurrent.AddMachine("auth", authFSM)
	concurrent.AddMachine("payment", paymentFSM)

	fmt.Printf("初始状态: %v\n", concurrent.GetStates())

	// 触发认证
	ctx := context.Background()
	concurrent.Trigger(ctx, "auth", "login")
	fmt.Printf("登录后: %v\n", concurrent.GetStates())

	// 触发支付
	concurrent.Trigger(ctx, "payment", "pay")
	fmt.Printf("支付后: %v\n\n", concurrent.GetStates())
}

// asyncExample 异步状态机示例
func asyncExample() {
	fmt.Println("4. 异步状态机示例")
	fmt.Println("-------------------")

	asyncFSM := statemachine.NewAsyncFSM("idle", 10)

	asyncFSM.AddTransition("idle", "processing", "process")
	asyncFSM.AddTransition("processing", "completed", "complete")

	asyncFSM.SetOnEnter("processing", func(ctx context.Context, state statemachine.State) error {
		fmt.Println("  → 开始处理任务")
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	asyncFSM.SetOnEnter("completed", func(ctx context.Context, state statemachine.State) error {
		fmt.Println("  → 任务完成")
		return nil
	})

	// 启动异步处理
	asyncFSM.Start()
	defer asyncFSM.Stop()

	ctx := context.Background()

	// 异步触发事件
	fmt.Println("提交异步任务...")
	asyncFSM.TriggerAsync(ctx, "process")
	asyncFSM.TriggerAsync(ctx, "complete")

	// 等待处理完成
	time.Sleep(300 * time.Millisecond)
	fmt.Printf("最终状态: %s\n", asyncFSM.Current())
}
