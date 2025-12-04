package statemachine

import (
	"context"
	"testing"
)

func TestHSM_BasicHierarchy(t *testing.T) {
	hsm := NewHSM("root")
	hsm.AddState("working", "root")
	hsm.AddState("coding", "working")

	_ = hsm.AddTransition("root", "working", "start_work")
	_ = hsm.AddTransition("working", "coding", "start_coding")

	ctx := context.Background()

	err := hsm.Trigger(ctx, "start_work")
	if err != nil {
		t.Fatalf("触发事件失败: %v", err)
	}

	if hsm.Current() != "working" {
		t.Errorf("状态错误: got %v, want working", hsm.Current())
	}

	err = hsm.Trigger(ctx, "start_coding")
	if err != nil {
		t.Fatalf("触发事件失败: %v", err)
	}

	if hsm.Current() != "coding" {
		t.Errorf("状态错误: got %v, want coding", hsm.Current())
	}
}

func TestHSM_EventInheritance(t *testing.T) {
	hsm := NewHSM("root")
	hsm.AddState("working", "root")
	hsm.AddState("coding", "working")

	// 在父状态定义转换
	_ = hsm.AddTransition("working", "root", "stop")

	// 切换到子状态
	_ = hsm.AddTransition("root", "coding", "start")
	ctx := context.Background()
	_ = hsm.Trigger(ctx, "start")

	// 子状态应该能继承父状态的转换
	if !hsm.Can("stop") {
		t.Error("子状态应该能继承父状态的事件")
	}

	err := hsm.Trigger(ctx, "stop")
	if err != nil {
		t.Fatalf("触发继承事件失败: %v", err)
	}

	if hsm.Current() != "root" {
		t.Errorf("状态错误: got %v, want root", hsm.Current())
	}
}

func TestHSM_Callbacks(t *testing.T) {
	hsm := NewHSM("root")
	hsm.AddState("working", "root")
	hsm.AddState("idle", "root")
	_ = hsm.AddTransition("root", "working", "start")
	_ = hsm.AddTransition("working", "idle", "stop")

	entered := false
	exited := false

	hsm.SetOnEnter("idle", func(ctx context.Context, state State) error {
		entered = true
		return nil
	})

	hsm.SetOnExit("working", func(ctx context.Context, state State) error {
		exited = true
		return nil
	})

	ctx := context.Background()
	_ = hsm.Trigger(ctx, "start")
	_ = hsm.Trigger(ctx, "stop")

	if !entered {
		t.Error("OnEnter callback not called")
	}
	if !exited {
		t.Error("OnExit callback not called")
	}
}

func TestHSM_Reset(t *testing.T) {
	hsm := NewHSM("root")
	hsm.AddState("working", "root")
	_ = hsm.AddTransition("root", "working", "start")

	ctx := context.Background()
	_ = hsm.Trigger(ctx, "start")

	if hsm.Current() != "working" {
		t.Errorf("状态错误: got %v, want working", hsm.Current())
	}

	_ = hsm.Reset()

	if hsm.Current() != "root" {
		t.Errorf("重置后状态错误: got %v, want root", hsm.Current())
	}
}
