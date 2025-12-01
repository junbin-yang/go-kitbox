package statemachine

import (
	"context"
	"testing"
)

func TestConcurrent_AddRemove(t *testing.T) {
	concurrent := NewConcurrent()

	fsm1 := NewFSM("idle")
	fsm2 := NewFSM("stopped")

	concurrent.AddMachine("machine1", fsm1)
	concurrent.AddMachine("machine2", fsm2)

	if concurrent.Count() != 2 {
		t.Errorf("状态机数量错误: got %d, want 2", concurrent.Count())
	}

	concurrent.RemoveMachine("machine1")

	if concurrent.Count() != 1 {
		t.Errorf("状态机数量错误: got %d, want 1", concurrent.Count())
	}
}

func TestConcurrent_GetStates(t *testing.T) {
	concurrent := NewConcurrent()

	fsm1 := NewFSM("idle")
	fsm2 := NewFSM("stopped")

	concurrent.AddMachine("machine1", fsm1)
	concurrent.AddMachine("machine2", fsm2)

	states := concurrent.GetStates()

	if states["machine1"] != "idle" {
		t.Errorf("machine1 状态错误: got %v, want idle", states["machine1"])
	}

	if states["machine2"] != "stopped" {
		t.Errorf("machine2 状态错误: got %v, want stopped", states["machine2"])
	}
}

func TestConcurrent_TriggerAll(t *testing.T) {
	concurrent := NewConcurrent()

	fsm1 := NewFSM("idle")
	_ = fsm1.AddTransition("idle", "running", "start")

	fsm2 := NewFSM("idle")
	_ = fsm2.AddTransition("idle", "running", "start")

	concurrent.AddMachine("machine1", fsm1)
	concurrent.AddMachine("machine2", fsm2)

	ctx := context.Background()
	results := concurrent.TriggerAll(ctx, "start")

	if results["machine1"] != nil {
		t.Errorf("machine1 触发失败: %v", results["machine1"])
	}

	if results["machine2"] != nil {
		t.Errorf("machine2 触发失败: %v", results["machine2"])
	}

	states := concurrent.GetStates()
	if states["machine1"] != "running" || states["machine2"] != "running" {
		t.Error("状态转换失败")
	}
}

func TestConcurrent_GetMachine(t *testing.T) {
	concurrent := NewConcurrent()

	fsm := NewFSM("idle")
	concurrent.AddMachine("machine1", fsm)

	retrieved, ok := concurrent.GetMachine("machine1")
	if !ok {
		t.Error("GetMachine returned false")
	}
	if retrieved == nil {
		t.Error("GetMachine returned nil")
	}
	if retrieved.Current() != "idle" {
		t.Errorf("Expected state 'idle', got '%s'", retrieved.Current())
	}

	// Test non-existent machine
	_, ok = concurrent.GetMachine("nonexistent")
	if ok {
		t.Error("Expected false for non-existent machine")
	}
}

func TestConcurrent_Trigger(t *testing.T) {
	concurrent := NewConcurrent()

	fsm := NewFSM("idle")
	_ = fsm.AddTransition("idle", "running", "start")
	concurrent.AddMachine("machine1", fsm)

	ctx := context.Background()
	err := concurrent.Trigger(ctx, "machine1", "start")
	if err != nil {
		t.Errorf("Trigger failed: %v", err)
	}

	states := concurrent.GetStates()
	if states["machine1"] != "running" {
		t.Errorf("Expected state 'running', got '%s'", states["machine1"])
	}
}

func TestConcurrent_ResetAll(t *testing.T) {
	concurrent := NewConcurrent()

	fsm1 := NewFSM("idle")
	_ = fsm1.AddTransition("idle", "running", "start")
	fsm2 := NewFSM("stopped")
	_ = fsm2.AddTransition("stopped", "running", "start")

	concurrent.AddMachine("machine1", fsm1)
	concurrent.AddMachine("machine2", fsm2)

	// Trigger transitions
	ctx := context.Background()
	_ = concurrent.Trigger(ctx, "machine1", "start")
	_ = concurrent.Trigger(ctx, "machine2", "start")

	// Reset all
	concurrent.ResetAll()

	states := concurrent.GetStates()
	if states["machine1"] != "idle" {
		t.Errorf("Expected machine1 reset to 'idle', got '%s'", states["machine1"])
	}
	if states["machine2"] != "stopped" {
		t.Errorf("Expected machine2 reset to 'stopped', got '%s'", states["machine2"])
	}
}
