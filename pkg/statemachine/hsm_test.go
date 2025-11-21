package statemachine

import (
	"context"
	"testing"
)

func TestHSM_BasicHierarchy(t *testing.T) {
	hsm := NewHSM("root")
	hsm.AddState("working", "root")
	hsm.AddState("coding", "working")

	hsm.AddTransition("root", "working", "start_work")
	hsm.AddTransition("working", "coding", "start_coding")

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
	hsm.AddTransition("working", "root", "stop")

	// 切换到子状态
	hsm.AddTransition("root", "coding", "start")
	ctx := context.Background()
	hsm.Trigger(ctx, "start")

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
