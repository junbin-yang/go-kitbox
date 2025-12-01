package statemachine

import (
	"context"
	"testing"
	"time"
)

func TestNewAsyncFSM(t *testing.T) {
	fsm := NewAsyncFSM("idle", 10)
	if fsm == nil {
		t.Fatal("NewAsyncFSM returned nil")
	}
	if fsm.Current() != "idle" {
		t.Errorf("Expected initial state 'idle', got '%s'", fsm.Current())
	}
}

func TestAsyncFSMStartStop(t *testing.T) {
	fsm := NewAsyncFSM("idle", 10)
	fsm.Start()
	time.Sleep(10 * time.Millisecond)
	fsm.Stop()
}

func TestAsyncFSMTriggerAsync(t *testing.T) {
	fsm := NewAsyncFSM("idle", 10)
	_ = fsm.AddTransition("idle", "running", "start")
	_ = fsm.AddTransition("running", "idle", "stop")

	fsm.Start()
	defer fsm.Stop()

	ctx := context.Background()
	err := fsm.TriggerAsync(ctx, "start")
	if err != nil {
		t.Errorf("TriggerAsync failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if fsm.Current() != "running" {
		t.Errorf("Expected state 'running', got '%s'", fsm.Current())
	}
}

func TestAsyncFSMQueueLength(t *testing.T) {
	fsm := NewAsyncFSM("idle", 10)
	_ = fsm.AddTransition("idle", "running", "start")

	ctx := context.Background()
	_ = fsm.TriggerAsync(ctx, "start")

	length := fsm.QueueLength()
	if length < 0 {
		t.Errorf("Expected non-negative queue length, got %d", length)
	}
}

func TestAsyncFSMContextCancellation(t *testing.T) {
	fsm := NewAsyncFSM("idle", 0) // Zero buffer to test blocking
	_ = fsm.AddTransition("idle", "running", "start")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := fsm.TriggerAsync(ctx, "start")
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestAsyncFSMMultipleEvents(t *testing.T) {
	fsm := NewAsyncFSM("idle", 10)
	_ = fsm.AddTransition("idle", "running", "start")
	_ = fsm.AddTransition("running", "paused", "pause")
	_ = fsm.AddTransition("paused", "running", "resume")
	_ = fsm.AddTransition("running", "idle", "stop")

	fsm.Start()
	defer fsm.Stop()

	ctx := context.Background()
	_ = fsm.TriggerAsync(ctx, "start")
	_ = fsm.TriggerAsync(ctx, "pause")
	_ = fsm.TriggerAsync(ctx, "resume")
	_ = fsm.TriggerAsync(ctx, "stop")

	time.Sleep(100 * time.Millisecond)

	if fsm.Current() != "idle" {
		t.Errorf("Expected final state 'idle', got '%s'", fsm.Current())
	}
}
