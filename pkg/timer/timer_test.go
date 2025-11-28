package timer

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// 场景1：创建周期性定时器
func TestScenario1_CreatePeriodicTimer(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	var count int32
	err := mgr.CreateTimer("test", 100*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	if err != nil {
		t.Fatalf("创建定时器失败: %v", err)
	}

	time.Sleep(350 * time.Millisecond)

	finalCount := atomic.LoadInt32(&count)
	if finalCount < 3 || finalCount > 4 {
		t.Errorf("期望执行3-4次，实际 %d 次", finalCount)
	}
}

// 场景2：创建一次性定时器
func TestScenario2_CreateOnceTimer(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	var executed int32
	err := mgr.CreateOnceTimer("once", 100*time.Millisecond, func() {
		atomic.AddInt32(&executed, 1)
	})

	if err != nil {
		t.Fatalf("创建一次性定时器失败: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&executed) != 0 {
		t.Error("定时器不应在延迟前执行")
	}

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("期望执行1次，实际 %d 次", atomic.LoadInt32(&executed))
	}

	time.Sleep(200 * time.Millisecond)
	if atomic.LoadInt32(&executed) != 1 {
		t.Error("一次性定时器不应执行多次")
	}
}

// 场景3：停止定时器
func TestScenario3_RemoveTimer(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	var count int32
	_ = mgr.CreateTimer("test", 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	time.Sleep(120 * time.Millisecond)
	beforeStop := atomic.LoadInt32(&count)

	err := mgr.RemoveTimer("test")
	if err != nil {
		t.Fatalf("停止定时器失败: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	afterStop := atomic.LoadInt32(&count)

	if afterStop != beforeStop {
		t.Errorf("定时器停止后不应继续执行，停止前 %d 次，停止后 %d 次", beforeStop, afterStop)
	}
}

// 场景4：重置定时器间隔
func TestScenario4_ResetTimer(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	var count int32
	_ = mgr.CreateTimer("test", 100*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	time.Sleep(250 * time.Millisecond)
	countBefore := atomic.LoadInt32(&count)

	err := mgr.ResetTimer("test", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("重置定时器失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	countAfter := atomic.LoadInt32(&count)

	if countAfter <= countBefore {
		t.Error("重置后定时器应以新间隔执行")
	}
}

// 场景5：重置一次性定时器
func TestScenario5_ResetOnceTimer(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	var executed int32
	_ = mgr.CreateOnceTimer("once", 500*time.Millisecond, func() {
		atomic.AddInt32(&executed, 1)
	})

	time.Sleep(200 * time.Millisecond)
	err := mgr.ResetTimer("once", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("重置一次性定时器失败: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("期望执行1次，实际 %d 次", atomic.LoadInt32(&executed))
	}
}

// 场景6：获取定时器信息
func TestScenario6_GetTimer(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	_ = mgr.CreateTimer("test", 100*time.Millisecond, func() {})

	info, exists := mgr.GetTimer("test")
	if !exists {
		t.Fatal("定时器应存在")
	}

	if info.ID != "test" {
		t.Errorf("期望ID test，实际 %s", info.ID)
	}

	if info.Interval != 100*time.Millisecond {
		t.Errorf("期望间隔 100ms，实际 %v", info.Interval)
	}

	if info.IsOnce {
		t.Error("应为周期性定时器")
	}
}

// 场景7：列出所有定时器
func TestScenario7_ListTimers(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	_ = mgr.CreateTimer("timer1", 100*time.Millisecond, func() {})
	_ = mgr.CreateTimer("timer2", 200*time.Millisecond, func() {})
	_ = mgr.CreateOnceTimer("once1", time.Second, func() {})

	ids := mgr.ListTimers()
	if len(ids) != 3 {
		t.Errorf("期望3个定时器，实际 %d 个", len(ids))
	}

	_ = mgr.RemoveTimer("timer1")
	ids = mgr.ListTimers()
	if len(ids) != 2 {
		t.Errorf("移除后期望2个定时器，实际 %d 个", len(ids))
	}
}

// 场景8：获取定时器数量
func TestScenario8_GetTimerCount(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	if mgr.GetTimerCount() != 0 {
		t.Error("初始应无定时器")
	}

	mgr.CreateTimer("t1", 100*time.Millisecond, func() {})
	mgr.CreateTimer("t2", 100*time.Millisecond, func() {})

	if mgr.GetTimerCount() != 2 {
		t.Errorf("期望2个定时器，实际 %d 个", mgr.GetTimerCount())
	}
}

// 场景9：停止所有定时器
func TestScenario9_StopAll(t *testing.T) {
	mgr := NewManager()

	var count1, count2 int32
	mgr.CreateTimer("t1", 50*time.Millisecond, func() {
		atomic.AddInt32(&count1, 1)
	})
	mgr.CreateTimer("t2", 50*time.Millisecond, func() {
		atomic.AddInt32(&count2, 1)
	})

	time.Sleep(120 * time.Millisecond)
	mgr.StopAll()

	before1 := atomic.LoadInt32(&count1)
	before2 := atomic.LoadInt32(&count2)

	time.Sleep(150 * time.Millisecond)

	if atomic.LoadInt32(&count1) != before1 || atomic.LoadInt32(&count2) != before2 {
		t.Error("StopAll后所有定时器应停止")
	}
}

// 场景10：重复ID应失败
func TestScenario10_DuplicateID(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	err1 := mgr.CreateTimer("dup", 100*time.Millisecond, func() {})
	if err1 != nil {
		t.Fatalf("首次创建失败: %v", err1)
	}

	err2 := mgr.CreateTimer("dup", 200*time.Millisecond, func() {})
	if err2 == nil {
		t.Error("重复ID应返回错误")
	}
}

// 场景11：ScheduleFunc 解析时间规格
func TestScenario11_ScheduleFunc(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	var count int32
	err := mgr.ScheduleFunc("test", "100ms", func() {
		atomic.AddInt32(&count, 1)
	})

	if err != nil {
		t.Fatalf("ScheduleFunc失败: %v", err)
	}

	time.Sleep(250 * time.Millisecond)
	if atomic.LoadInt32(&count) < 2 {
		t.Error("定时器应执行至少2次")
	}
}

// 场景12：ScheduleFunc 无效格式
func TestScenario12_ScheduleFuncInvalid(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	err := mgr.ScheduleFunc("test", "invalid", func() {})
	if err == nil {
		t.Error("无效时间格式应返回错误")
	}
}

// 场景13：防抖函数
func TestScenario13_Debounce(t *testing.T) {
	var count int32
	debounced := Debounce(100*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	debounced()
	time.Sleep(50 * time.Millisecond)
	debounced()
	time.Sleep(50 * time.Millisecond)
	debounced()

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&count) != 0 {
		t.Error("防抖期间不应执行")
	}

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("期望执行1次，实际 %d 次", atomic.LoadInt32(&count))
	}
}

// 场景14：节流函数
func TestScenario14_Throttle(t *testing.T) {
	var count int32
	throttled := Throttle(100*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	throttled() // 执行
	throttled() // 忽略
	throttled() // 忽略

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("期望执行1次，实际 %d 次", atomic.LoadInt32(&count))
	}

	time.Sleep(150 * time.Millisecond)
	throttled() // 执行

	if atomic.LoadInt32(&count) != 2 {
		t.Errorf("期望执行2次，实际 %d 次", atomic.LoadInt32(&count))
	}
}

// 场景15：重试成功
func TestScenario15_RetrySuccess(t *testing.T) {
	attempt := 0
	err := Retry(3, 50*time.Millisecond, func() error {
		attempt++
		if attempt < 2 {
			return errors.New("失败")
		}
		return nil
	})

	if err != nil {
		t.Errorf("应成功，实际错误: %v", err)
	}

	if attempt != 2 {
		t.Errorf("期望尝试2次，实际 %d 次", attempt)
	}
}

// 场景16：重试失败
func TestScenario16_RetryFail(t *testing.T) {
	attempt := 0
	err := Retry(3, 50*time.Millisecond, func() error {
		attempt++
		return errors.New("持续失败")
	})

	if err == nil {
		t.Error("应返回错误")
	}

	if attempt != 3 {
		t.Errorf("期望尝试3次，实际 %d 次", attempt)
	}
}

// 场景17：指数退避重试
func TestScenario17_ExponentialBackoff(t *testing.T) {
	attempt := 0
	start := time.Now()

	err := ExponentialBackoff(3, 50*time.Millisecond, func() error {
		attempt++
		if attempt < 3 {
			return errors.New("失败")
		}
		return nil
	})

	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("应成功，实际错误: %v", err)
	}

	// 第1次立即，第2次等50ms，第3次等100ms，总计至少150ms
	if elapsed < 150*time.Millisecond {
		t.Errorf("指数退避时间不足，实际 %v", elapsed)
	}
}

// 场景18：回调panic不应崩溃
func TestScenario18_CallbackPanic(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	var count int32
	err := mgr.CreateTimer("panic", 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
		panic("测试panic")
	})

	if err != nil {
		t.Fatalf("创建定时器失败: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 即使panic，定时器应继续执行
	if atomic.LoadInt32(&count) < 3 {
		t.Error("panic后定时器应继续执行")
	}
}

// 场景19：一次性定时器自动移除
func TestScenario19_OnceTimerAutoRemove(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	_ = mgr.CreateOnceTimer("once", 100*time.Millisecond, func() {})

	if mgr.GetTimerCount() != 1 {
		t.Error("创建后应有1个定时器")
	}

	time.Sleep(200 * time.Millisecond)

	if mgr.GetTimerCount() != 0 {
		t.Error("执行后一次性定时器应自动移除")
	}
}

// 场景20：并发创建定时器
func TestScenario20_ConcurrentCreate(t *testing.T) {
	mgr := NewManager()
	defer mgr.StopAll()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			err := mgr.CreateTimer(string(rune('a'+id)), 100*time.Millisecond, func() {})
			if err != nil {
				t.Errorf("并发创建失败: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if mgr.GetTimerCount() != 10 {
		t.Errorf("期望10个定时器，实际 %d 个", mgr.GetTimerCount())
	}
}
