# StateMachine - 状态机工具库

提供多种类型的状态机实现，包括有限状态机(FSM)、层次状态机(HSM)、并发状态机等，面向通用业务场景。

## 特性

- **有限状态机(FSM)** - 基础状态机，支持状态转换和事件触发
- **层次状态机(HSM)** - 支持状态嵌套和继承
- **并发状态机** - 支持多个状态机并行运行
- **异步事件处理** - 支持事件队列和异步触发
- **状态守卫** - 转换前的条件检查
- **状态钩子** - 进入/退出/转换时的回调
- **状态持久化** - 状态保存和恢复
- **并发安全** - 所有操作都是线程安全的

## 安装

```bash
go get github.com/junbin-yang/go-kitbox/pkg/statemachine
```

## 快速开始

### 基础有限状态机

```go
package main

import (
    "context"
    "fmt"
    "github.com/junbin-yang/go-kitbox/pkg/statemachine"
)

func main() {
    // 创建状态机
    fsm := statemachine.NewFSM("idle")

    // 添加状态转换
    fsm.AddTransition("idle", "running", "start")
    fsm.AddTransition("running", "idle", "stop")

    // 触发事件
    ctx := context.Background()
    fsm.Trigger(ctx, "start")

    fmt.Println(fsm.Current()) // 输出: running
}
```

## 核心功能

### 1. 有限状态机 (FSM)

基础状态机实现，支持状态转换、守卫条件和回调函数。

```go
fsm := statemachine.NewFSM("pending")

// 添加转换规则
fsm.AddTransition("pending", "paid", "pay")
fsm.AddTransition("paid", "shipped", "ship")

// 设置进入状态回调
fsm.SetOnEnter("paid", func(ctx context.Context, state statemachine.State) error {
    fmt.Println("订单已支付")
    return nil
})

// 设置退出状态回调
fsm.SetOnExit("pending", func(ctx context.Context, state statemachine.State) error {
    fmt.Println("离开待支付状态")
    return nil
})

// 触发事件
ctx := context.Background()
fsm.Trigger(ctx, "pay")
```

### 2. 状态守卫

在状态转换前检查条件。

```go
fsm := statemachine.NewFSM("idle")

// 添加带守卫的转换
guard := func(ctx context.Context, from, to statemachine.State) bool {
    // 检查是否满足转换条件
    return checkPermission()
}

fsm.AddTransitionWithGuard("idle", "running", "start", guard)

ctx := context.Background()
err := fsm.Trigger(ctx, "start")
if err == statemachine.ErrTransitionDenied {
    fmt.Println("转换被守卫拒绝")
}
```

### 3. 层次状态机 (HSM)

支持状态嵌套和继承，子状态可以继承父状态的转换规则。

```go
hsm := statemachine.NewHSM("root")

// 定义层次结构
hsm.AddState("working", "root")      // working 是 root 的子状态
hsm.AddState("coding", "working")    // coding 是 working 的子状态
hsm.AddState("testing", "working")   // testing 是 working 的子状态

// 添加转换
hsm.AddTransition("root", "working", "start_work")
hsm.AddTransition("working", "coding", "start_coding")
hsm.AddTransition("working", "root", "finish_work")

ctx := context.Background()
hsm.Trigger(ctx, "start_work")
hsm.Trigger(ctx, "start_coding")

// 子状态可以触发父状态的事件
hsm.Trigger(ctx, "finish_work") // 从 coding 直接回到 root
```

### 4. 并发状态机

管理多个状态机并行运行。

```go
concurrent := statemachine.NewConcurrent()

// 创建多个状态机
authFSM := statemachine.NewFSM("logged_out")
authFSM.AddTransition("logged_out", "logged_in", "login")

paymentFSM := statemachine.NewFSM("unpaid")
paymentFSM.AddTransition("unpaid", "paid", "pay")

// 添加到并发管理器
concurrent.AddMachine("auth", authFSM)
concurrent.AddMachine("payment", paymentFSM)

// 触发单个状态机
ctx := context.Background()
concurrent.Trigger(ctx, "auth", "login")

// 触发所有状态机
results := concurrent.TriggerAll(ctx, "reset")

// 获取所有状态
states := concurrent.GetStates()
fmt.Println(states) // map[auth:logged_in payment:unpaid]
```

### 5. 异步状态机

支持事件队列和异步处理。

```go
asyncFSM := statemachine.NewAsyncFSM("idle", 100) // 队列大小100

asyncFSM.AddTransition("idle", "processing", "process")
asyncFSM.AddTransition("processing", "completed", "complete")

// 启动异步处理
asyncFSM.Start()
defer asyncFSM.Stop()

// 异步触发事件
ctx := context.Background()
asyncFSM.TriggerAsync(ctx, "process")
asyncFSM.TriggerAsync(ctx, "complete")

// 事件会在后台队列中依次处理
```

### 6. 状态持久化

支持状态快照和历史记录。

```go
persistentFSM := statemachine.NewPersistentFSM("idle")

persistentFSM.AddTransition("idle", "running", "start")
persistentFSM.AddTransition("running", "idle", "stop")

ctx := context.Background()
persistentFSM.Trigger(ctx, "start")

// 创建快照
snapshot := persistentFSM.CreateSnapshot(map[string]interface{}{
    "user": "admin",
    "timestamp": time.Now(),
})

// 恢复快照
persistentFSM.RestoreSnapshot(snapshot)

// 获取历史记录
history := persistentFSM.GetHistory()
```

## 应用场景

### 场景 1: 订单状态管理

```go
orderFSM := statemachine.NewFSM("pending")

// 定义订单状态流转
orderFSM.AddTransition("pending", "paid", "pay")
orderFSM.AddTransition("paid", "shipped", "ship")
orderFSM.AddTransition("shipped", "delivered", "deliver")
orderFSM.AddTransition("delivered", "completed", "complete")

// 添加业务逻辑
orderFSM.SetOnEnter("paid", func(ctx context.Context, state statemachine.State) error {
    return sendPaymentNotification(ctx)
})

orderFSM.SetOnEnter("shipped", func(ctx context.Context, state statemachine.State) error {
    return updateInventory(ctx)
})
```

### 场景 2: 工作流引擎

```go
workflowHSM := statemachine.NewHSM("draft")

// 定义审批层次
workflowHSM.AddState("reviewing", "draft")
workflowHSM.AddState("manager_review", "reviewing")
workflowHSM.AddState("director_review", "reviewing")

workflowHSM.AddTransition("draft", "manager_review", "submit")
workflowHSM.AddTransition("manager_review", "director_review", "approve")
workflowHSM.AddTransition("director_review", "approved", "approve")
workflowHSM.AddTransition("reviewing", "rejected", "reject")
```

### 场景 3: 游戏角色状态

```go
characterFSM := statemachine.NewFSM("idle")

characterFSM.AddTransition("idle", "walking", "walk")
characterFSM.AddTransition("walking", "running", "run")
characterFSM.AddTransition("running", "jumping", "jump")
characterFSM.AddTransition("jumping", "idle", "land")

// 添加守卫检查体力
characterFSM.AddTransitionWithGuard("walking", "running", "run",
    func(ctx context.Context, from, to statemachine.State) bool {
        return character.Stamina > 20
    })
```

### 场景 4: 网络连接状态

```go
connFSM := statemachine.NewFSM("disconnected")

connFSM.AddTransition("disconnected", "connecting", "connect")
connFSM.AddTransition("connecting", "connected", "success")
connFSM.AddTransition("connecting", "disconnected", "fail")
connFSM.AddTransition("connected", "disconnected", "disconnect")

// 添加重连逻辑
connFSM.SetOnEnter("disconnected", func(ctx context.Context, state statemachine.State) error {
    return scheduleReconnect(ctx)
})
```

## API 参考

### FSM 方法

| 方法 | 说明 |
|------|------|
| `NewFSM(initial State)` | 创建有限状态机 |
| `AddTransition(from, to State, event Event)` | 添加状态转换 |
| `AddTransitionWithGuard(from, to State, event Event, guard GuardFunc)` | 添加带守卫的转换 |
| `SetOnEnter(state State, action ActionFunc)` | 设置进入状态回调 |
| `SetOnExit(state State, action ActionFunc)` | 设置退出状态回调 |
| `SetOnTransition(from State, event Event, fn TransitionFunc)` | 设置转换回调 |
| `Trigger(ctx Context, event Event)` | 触发事件 |
| `Can(event Event)` | 检查是否可以触发事件 |
| `Current()` | 获取当前状态 |
| `Reset()` | 重置到初始状态 |

### HSM 方法

| 方法 | 说明 |
|------|------|
| `NewHSM(initial State)` | 创建层次状态机 |
| `AddState(child, parent State)` | 添加子状态 |
| 其他方法同 FSM | |

### Concurrent 方法

| 方法 | 说明 |
|------|------|
| `NewConcurrent()` | 创建并发管理器 |
| `AddMachine(name string, machine StateMachine)` | 添加状态机 |
| `RemoveMachine(name string)` | 移除状态机 |
| `GetMachine(name string)` | 获取状态机 |
| `Trigger(ctx Context, name string, event Event)` | 触发指定状态机 |
| `TriggerAll(ctx Context, event Event)` | 触发所有状态机 |
| `GetStates()` | 获取所有状态 |
| `ResetAll()` | 重置所有状态机 |
| `Count()` | 获取状态机数量 |

### AsyncFSM 方法

| 方法 | 说明 |
|------|------|
| `NewAsyncFSM(initial State, queueSize int)` | 创建异步状态机 |
| `Start()` | 启动异步处理 |
| `Stop()` | 停止异步处理 |
| `TriggerAsync(ctx Context, event Event)` | 异步触发事件 |
| `QueueLength()` | 获取队列长度 |

## 最佳实践

1. **选择合适的状态机类型**
   - 简单状态流转 → FSM
   - 有层次关系 → HSM
   - 多个独立状态机 → Concurrent
   - 高并发场景 → AsyncFSM

2. **使用守卫条件**
   ```go
   fsm.AddTransitionWithGuard(from, to, event, func(ctx, from, to) bool {
       return validateBusinessRule(ctx)
   })
   ```

3. **合理使用回调**
   - OnEnter: 初始化资源、发送通知
   - OnExit: 清理资源、保存状态
   - OnTransition: 记录日志、更新数据

4. **错误处理**
   ```go
   err := fsm.Trigger(ctx, event)
   switch err {
   case statemachine.ErrInvalidTransition:
       // 处理无效转换
   case statemachine.ErrTransitionDenied:
       // 处理守卫拒绝
   }
   ```

5. **并发安全**
   - 所有状态机都是并发安全的
   - 回调函数应避免长时间阻塞
   - 耗时操作应在新 goroutine 中执行

## 许可证

MIT License
