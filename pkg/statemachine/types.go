package statemachine

import "context"

// State 表示状态机中的状态
type State string

// Event 表示触发状态转换的事件
type Event string

// TransitionFunc 在状态转换时调用
type TransitionFunc func(ctx context.Context, from, to State) error

// GuardFunc 检查是否允许状态转换
type GuardFunc func(ctx context.Context, from, to State) bool

// ActionFunc 在状态进入或退出时执行
type ActionFunc func(ctx context.Context, state State) error

// StateMachine 定义所有状态机的核心接口
type StateMachine interface {
	// Current 返回当前状态
	Current() State

	// Trigger 触发事件以转换状态
	Trigger(ctx context.Context, event Event) error

	// Can 检查是否可以从当前状态触发事件
	Can(event Event) bool

	// Reset 重置状态机到初始状态
	Reset() error
}
