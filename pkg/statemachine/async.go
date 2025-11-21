package statemachine

import (
	"context"
	"sync"
)

// AsyncEvent 异步事件
type AsyncEvent struct {
	Event   Event
	Context context.Context
}

// AsyncFSM 支持异步事件处理的状态机
type AsyncFSM struct {
	*FSM
	eventQueue chan AsyncEvent
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewAsyncFSM 创建异步状态机
func NewAsyncFSM(initial State, queueSize int) *AsyncFSM {
	return &AsyncFSM{
		FSM:        NewFSM(initial),
		eventQueue: make(chan AsyncEvent, queueSize),
		stopCh:     make(chan struct{}),
	}
}

// Start 启动异步事件处理
func (a *AsyncFSM) Start() {
	a.wg.Add(1)
	go a.processEvents()
}

// Stop 停止异步事件处理
func (a *AsyncFSM) Stop() {
	close(a.stopCh)
	a.wg.Wait()
}

// TriggerAsync 异步触发事件
func (a *AsyncFSM) TriggerAsync(ctx context.Context, event Event) error {
	select {
	case a.eventQueue <- AsyncEvent{Event: event, Context: ctx}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// processEvents 处理事件队列
func (a *AsyncFSM) processEvents() {
	defer a.wg.Done()

	for {
		select {
		case <-a.stopCh:
			return
		case asyncEvent := <-a.eventQueue:
			_ = a.FSM.Trigger(asyncEvent.Context, asyncEvent.Event)
		}
	}
}

// QueueLength 返回队列长度
func (a *AsyncFSM) QueueLength() int {
	return len(a.eventQueue)
}
