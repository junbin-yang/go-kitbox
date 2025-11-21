package statemachine

import (
	"context"
	"sync"
)

// Concurrent 并发状态机管理器
type Concurrent struct {
	mu       sync.RWMutex
	machines map[string]StateMachine
	running  map[string]bool
}

// NewConcurrent 创建并发状态机管理器
func NewConcurrent() *Concurrent {
	return &Concurrent{
		machines: make(map[string]StateMachine),
		running:  make(map[string]bool),
	}
}

// AddMachine 添加状态机
func (c *Concurrent) AddMachine(name string, machine StateMachine) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.machines[name] = machine
	c.running[name] = false
}

// RemoveMachine 移除状态机
func (c *Concurrent) RemoveMachine(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.machines, name)
	delete(c.running, name)
}

// GetMachine 获取状态机
func (c *Concurrent) GetMachine(name string) (StateMachine, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	machine, exists := c.machines[name]
	return machine, exists
}

// Trigger 触发指定状态机的事件
func (c *Concurrent) Trigger(ctx context.Context, name string, event Event) error {
	c.mu.RLock()
	machine, exists := c.machines[name]
	c.mu.RUnlock()

	if !exists {
		return ErrStateNotFound
	}

	return machine.Trigger(ctx, event)
}

// TriggerAll 触发所有状态机的相同事件
func (c *Concurrent) TriggerAll(ctx context.Context, event Event) map[string]error {
	c.mu.RLock()
	machines := make(map[string]StateMachine, len(c.machines))
	for name, machine := range c.machines {
		machines[name] = machine
	}
	c.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, machine := range machines {
		wg.Add(1)
		go func(n string, m StateMachine) {
			defer wg.Done()
			err := m.Trigger(ctx, event)
			mu.Lock()
			results[n] = err
			mu.Unlock()
		}(name, machine)
	}

	wg.Wait()
	return results
}

// GetStates 获取所有状态机的当前状态
func (c *Concurrent) GetStates() map[string]State {
	c.mu.RLock()
	defer c.mu.RUnlock()

	states := make(map[string]State, len(c.machines))
	for name, machine := range c.machines {
		states[name] = machine.Current()
	}
	return states
}

// ResetAll 重置所有状态机
func (c *Concurrent) ResetAll() map[string]error {
	c.mu.RLock()
	machines := make(map[string]StateMachine, len(c.machines))
	for name, machine := range c.machines {
		machines[name] = machine
	}
	c.mu.RUnlock()

	results := make(map[string]error)
	for name, machine := range machines {
		results[name] = machine.Reset()
	}
	return results
}

// Count 返回状态机数量
func (c *Concurrent) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.machines)
}
