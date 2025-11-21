package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/junbin-yang/go-kitbox/pkg/timer"
)

func main() {
	fmt.Println("=== Timer 示例 ===")
	fmt.Println()

	// 示例 1: 周期性定时器
	fmt.Println("1. 周期性定时器:")
	mgr := timer.NewManager()
	defer mgr.StopAll()

	count := 0
	mgr.CreateTimer("periodic", 500*time.Millisecond, func() {
		count++
		fmt.Printf("   周期性任务执行 #%d\n", count)
	})
	time.Sleep(1600 * time.Millisecond)
	fmt.Println()

	// 示例 2: 一次性定时器
	fmt.Println("2. 一次性定时器:")
	mgr.CreateOnceTimer("once", 800*time.Millisecond, func() {
		fmt.Println("   延迟任务执行")
	})
	time.Sleep(1 * time.Second)
	fmt.Println()

	// 示例 3: 查询定时器信息
	fmt.Println("3. 查询定时器信息:")
	info, exists := mgr.GetTimer("periodic")
	if exists {
		fmt.Printf("   ID: %s, 间隔: %v, 一次性: %v\n", info.ID, info.Interval, info.IsOnce)
	}
	fmt.Printf("   活跃定时器数量: %d\n", mgr.GetTimerCount())
	fmt.Printf("   定时器列表: %v\n", mgr.ListTimers())
	fmt.Println()

	// 示例 4: 重置定时器间隔
	fmt.Println("4. 重置定时器间隔:")
	mgr.ResetTimer("periodic", 300*time.Millisecond)
	fmt.Println("   已将间隔改为300ms")
	time.Sleep(1 * time.Second)
	fmt.Println()

	// 示例 5: 停止定时器
	fmt.Println("5. 停止定时器:")
	mgr.RemoveTimer("periodic")
	fmt.Println("   定时器已停止")
	time.Sleep(500 * time.Millisecond)
	fmt.Println()

	// 示例 6: 防抖函数
	fmt.Println("6. 防抖函数 (500ms):")
	searchCount := 0
	search := timer.Debounce(500*time.Millisecond, func() {
		searchCount++
		fmt.Printf("   执行搜索 #%d\n", searchCount)
	})

	fmt.Println("   模拟快速输入...")
	search()
	time.Sleep(200 * time.Millisecond)
	search()
	time.Sleep(200 * time.Millisecond)
	search()
	time.Sleep(700 * time.Millisecond)
	fmt.Println()

	// 示例 7: 节流函数
	fmt.Println("7. 节流函数 (500ms):")
	clickCount := 0
	onClick := timer.Throttle(500*time.Millisecond, func() {
		clickCount++
		fmt.Printf("   按钮点击 #%d\n", clickCount)
	})

	fmt.Println("   模拟快速点击...")
	onClick()
	onClick()
	onClick()
	time.Sleep(600 * time.Millisecond)
	onClick()
	onClick()
	time.Sleep(200 * time.Millisecond)
	fmt.Println()

	// 示例 8: 固定间隔重试
	fmt.Println("8. 固定间隔重试:")
	attempt := 0
	err := timer.Retry(3, 300*time.Millisecond, func() error {
		attempt++
		fmt.Printf("   尝试 #%d\n", attempt)
		if attempt < 3 {
			return errors.New("失败")
		}
		return nil
	})
	if err != nil {
		fmt.Printf("   最终失败: %v\n", err)
	} else {
		fmt.Println("   成功!")
	}
	fmt.Println()

	// 示例 9: 指数退避重试
	fmt.Println("9. 指数退避重试:")
	attempt2 := 0
	err2 := timer.ExponentialBackoff(4, 100*time.Millisecond, func() error {
		attempt2++
		fmt.Printf("   尝试 #%d\n", attempt2)
		if attempt2 < 3 {
			return errors.New("网络错误")
		}
		return nil
	})
	if err2 != nil {
		fmt.Printf("   最终失败: %v\n", err2)
	} else {
		fmt.Println("   成功!")
	}
	fmt.Println()

	// 示例 10: 字符串解析定时器
	fmt.Println("10. 字符串解析定时器:")
	scheduleCount := 0
	mgr.ScheduleFunc("scheduled", "400ms", func() {
		scheduleCount++
		fmt.Printf("   定时任务 #%d\n", scheduleCount)
	})
	time.Sleep(1300 * time.Millisecond)

	fmt.Println()
	fmt.Println("示例完成!")
}
