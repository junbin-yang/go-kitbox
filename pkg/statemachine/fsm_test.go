package statemachine

import (
	"context"
	"testing"
)

func TestFSM_BasicTransition(t *testing.T) {
	fsm := NewFSM("idle")

	err := fsm.AddTransition("idle", "running", "start")
	if err != nil {
		t.Fatalf("添加转换失败: %v", err)
	}

	err = fsm.AddTransition("running", "idle", "stop")
	if err != nil {
		t.Fatalf("添加转换失败: %v", err)
	}

	if fsm.Current() != "idle" {
		t.Errorf("初始状态错误: got %v, want idle", fsm.Current())
	}

	ctx := context.Background()
	err = fsm.Trigger(ctx, "start")
	if err != nil {
		t.Fatalf("触发事件失败: %v", err)
	}

	if fsm.Current() != "running" {
		t.Errorf("状态转换失败: got %v, want running", fsm.Current())
	}
}

func TestFSM_InvalidTransition(t *testing.T) {
	fsm := NewFSM("idle")
	_ = fsm.AddTransition("idle", "running", "start")

	ctx := context.Background()
	err := fsm.Trigger(ctx, "invalid")
	if err != ErrInvalidTransition {
		t.Errorf("期望 ErrInvalidTransition, got %v", err)
	}
}

func TestFSM_Guard(t *testing.T) {
	fsm := NewFSM("idle")

	guard := func(ctx context.Context, from, to State) bool {
		return false
	}

	err := fsm.AddTransitionWithGuard("idle", "running", "start", guard)
	if err != nil {
		t.Fatalf("添加转换失败: %v", err)
	}

	ctx := context.Background()
	err = fsm.Trigger(ctx, "start")
	if err != ErrTransitionDenied {
		t.Errorf("期望 ErrTransitionDenied, got %v", err)
	}
}

func TestFSM_Callbacks(t *testing.T) {
	fsm := NewFSM("idle")
	_ = fsm.AddTransition("idle", "running", "start")

	entered := false
	exited := false

	fsm.SetOnEnter("running", func(ctx context.Context, state State) error {
		entered = true
		return nil
	})

	fsm.SetOnExit("idle", func(ctx context.Context, state State) error {
		exited = true
		return nil
	})

	ctx := context.Background()
	_ = fsm.Trigger(ctx, "start")

	if !entered {
		t.Error("OnEnter 回调未执行")
	}
	if !exited {
		t.Error("OnExit 回调未执行")
	}
}

func TestFSM_Reset(t *testing.T) {
	fsm := NewFSM("idle")
	_ = fsm.AddTransition("idle", "running", "start")

	ctx := context.Background()
	_ = fsm.Trigger(ctx, "start")

	if fsm.Current() != "running" {
		t.Errorf("状态错误: got %v, want running", fsm.Current())
	}

	_ = fsm.Reset()

	if fsm.Current() != "idle" {
		t.Errorf("重置后状态错误: got %v, want idle", fsm.Current())
	}
}

func TestFSM_Can(t *testing.T) {
	fsm := NewFSM("idle")
	_ = fsm.AddTransition("idle", "running", "start")

	if !fsm.Can("start") {
		t.Error("应该可以触发 start 事件")
	}

	if fsm.Can("stop") {
		t.Error("不应该可以触发 stop 事件")
	}
}
