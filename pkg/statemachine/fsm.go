package statemachine

import (
	"context"
	"sync"
)

// FSM 有限状态机实现
type FSM struct {
	mu           sync.RWMutex
	current      State
	initial      State
	transitions  map[transitionKey]*Transition
	onEnter      map[State]ActionFunc
	onExit       map[State]ActionFunc
}

// NewFSM 创建新的有限状态机
func NewFSM(initial State) *FSM {
	return &FSM{
		current:     initial,
		initial:     initial,
		transitions: make(map[transitionKey]*Transition),
		onEnter:     make(map[State]ActionFunc),
		onExit:      make(map[State]ActionFunc),
	}
}

// Current 返回当前状态
func (f *FSM) Current() State {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

// AddTransition 添加状态转换规则
func (f *FSM) AddTransition(from, to State, event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := transitionKey{from: from, event: event}
	if _, exists := f.transitions[key]; exists {
		return ErrDuplicateTransition
	}

	f.transitions[key] = &Transition{
		From:  from,
		To:    to,
		Event: event,
	}
	return nil
}

// AddTransitionWithGuard 添加带守卫的状态转换规则
func (f *FSM) AddTransitionWithGuard(from, to State, event Event, guard GuardFunc) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := transitionKey{from: from, event: event}
	if _, exists := f.transitions[key]; exists {
		return ErrDuplicateTransition
	}

	f.transitions[key] = &Transition{
		From:  from,
		To:    to,
		Event: event,
		Guard: guard,
	}
	return nil
}

// SetOnEnter 设置状态进入时的回调
func (f *FSM) SetOnEnter(state State, action ActionFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onEnter[state] = action
}

// SetOnExit 设置状态退出时的回调
func (f *FSM) SetOnExit(state State, action ActionFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onExit[state] = action
}

// SetOnTransition 设置转换时的回调
func (f *FSM) SetOnTransition(from State, event Event, fn TransitionFunc) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := transitionKey{from: from, event: event}
	trans, exists := f.transitions[key]
	if !exists {
		return ErrEventNotFound
	}

	trans.OnTransition = fn
	return nil
}

// Can 检查是否可以触发事件
func (f *FSM) Can(event Event) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	key := transitionKey{from: f.current, event: event}
	_, exists := f.transitions[key]
	return exists
}

// Trigger 触发事件进行状态转换
func (f *FSM) Trigger(ctx context.Context, event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := transitionKey{from: f.current, event: event}
	trans, exists := f.transitions[key]
	if !exists {
		return ErrInvalidTransition
	}

	// 检查守卫条件
	if trans.Guard != nil && !trans.Guard(ctx, f.current, trans.To) {
		return ErrTransitionDenied
	}

	from := f.current

	// 执行退出回调
	if exitFn, ok := f.onExit[from]; ok {
		if err := exitFn(ctx, from); err != nil {
			return err
		}
	}

	// 执行转换回调
	if trans.OnTransition != nil {
		if err := trans.OnTransition(ctx, from, trans.To); err != nil {
			return err
		}
	}

	// 更新状态
	f.current = trans.To

	// 执行进入回调
	if enterFn, ok := f.onEnter[trans.To]; ok {
		if err := enterFn(ctx, trans.To); err != nil {
			return err
		}
	}

	return nil
}

// Reset 重置到初始状态
func (f *FSM) Reset() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = f.initial
	return nil
}
