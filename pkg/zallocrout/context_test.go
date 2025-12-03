package zallocrout

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type testContextKey string

const (
	testParentKey testContextKey = "parent_key"
	testRouteKey  testContextKey = "route_key"
)

// TestRouteContext_GetParam 测试 GetParam 方法
func TestRouteContext_GetParam(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "123"}
	paramPairs[1] = paramPair{key: "name", value: "test"}
	paramCount := 2

	ctx := acquireContext(parent, &paramPairs, paramCount)
	defer releaseContext(ctx)

	// 测试已存在的参数
	val, ok := ctx.GetParam("id")
	if !ok || val != "123" {
		t.Errorf("GetParam(id) = %v, %v; want 123, true", val, ok)
	}

	val, ok = ctx.GetParam("name")
	if !ok || val != "test" {
		t.Errorf("GetParam(name) = %v, %v; want test, true", val, ok)
	}

	// 测试不存在的参数
	val, ok = ctx.GetParam("nonexistent")
	if ok || val != "" {
		t.Errorf("GetParam(nonexistent) = %v, %v; want '', false", val, ok)
	}
}

// TestRouteContext_SetValue 测试 SetValue 方法
func TestRouteContext_SetValue(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// 测试设置值
	ok := ctx.SetValue("key1", "value1")
	if !ok {
		t.Error("SetValue(key1) failed")
	}

	ok = ctx.SetValue("key2", 42)
	if !ok {
		t.Error("SetValue(key2) failed")
	}

	// 测试读取值
	val, ok := ctx.GetValue("key1")
	if !ok || val != "value1" {
		t.Errorf("GetValue(key1) = %v, %v; want value1, true", val, ok)
	}

	val, ok = ctx.GetValue("key2")
	if !ok || val != 42 {
		t.Errorf("GetValue(key2) = %v, %v; want 42, true", val, ok)
	}

	// 测试不存在的值
	val, ok = ctx.GetValue("nonexistent")
	if ok || val != nil {
		t.Errorf("GetValue(nonexistent) = %v, %v; want nil, false", val, ok)
	}
}

// TestRouteContext_SetValue_MaxValues 测试 MaxValues 限制
func TestRouteContext_SetValue_MaxValues(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// 填充到 MaxValues
	for i := 0; i < MaxValues; i++ {
		ok := ctx.SetValue("key", i)
		if !ok {
			t.Errorf("SetValue failed at index %d", i)
		}
	}

	// 尝试超过 MaxValues
	ok := ctx.SetValue("overflow", "value")
	if ok {
		t.Error("SetValue should fail when MaxValues is exceeded")
	}
}

// TestRouteContext_Value 测试 Value 方法（context.Context 接口）
func TestRouteContext_Value(t *testing.T) {
	parent := context.WithValue(context.Background(), testParentKey, "parent_value")
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// 在 routeContext 中设置值
	ctx.SetValue(string(testRouteKey), "route_value")

	// 测试通过 Value 方法获取 routeContext 值
	val := ctx.Value(string(testRouteKey))
	if val != "route_value" {
		t.Errorf("Value(route_key) = %v; want route_value", val)
	}

	// 测试获取父 context 值
	val = ctx.Value(testParentKey)
	if val != "parent_value" {
		t.Errorf("Value(parent_key) = %v; want parent_value", val)
	}

	// 测试不存在的值
	val = ctx.Value(testContextKey("nonexistent"))
	if val != nil {
		t.Errorf("Value(nonexistent) = %v; want nil", val)
	}

	// 测试非字符串键
	val = ctx.Value(123)
	if val != nil {
		t.Errorf("Value(123) = %v; want nil", val)
	}
}

// TestRouteContext_Deadline 测试 Deadline 方法
func TestRouteContext_Deadline(t *testing.T) {
	// 测试没有 deadline 的父 context
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	deadline, ok := ctx.Deadline()
	if ok {
		t.Errorf("Deadline() = %v, %v; want zero, false", deadline, ok)
	}

	// 测试有 deadline 的父 context
	expectedDeadline := time.Now().Add(time.Hour)
	parentWithDeadline, cancel := context.WithDeadline(context.Background(), expectedDeadline)
	defer cancel()

	ctx2 := acquireContext(parentWithDeadline, &paramPairs, 0)
	defer releaseContext(ctx2)

	deadline, ok = ctx2.Deadline()
	if !ok {
		t.Error("Deadline() should return true for context with deadline")
	}
	if !deadline.Equal(expectedDeadline) {
		t.Errorf("Deadline() = %v; want %v", deadline, expectedDeadline)
	}
}

// TestRouteContext_Done 测试 Done 方法
func TestRouteContext_Done(t *testing.T) {
	// 测试不可取消的父 context
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	done := ctx.Done()
	if done != nil {
		t.Error("Done() should return nil for non-cancellable context")
	}

	// 测试可取消的父 context
	parentCtx, cancel := context.WithCancel(context.Background())
	ctx2 := acquireContext(parentCtx, &paramPairs, 0)
	defer releaseContext(ctx2)

	done = ctx2.Done()
	if done == nil {
		t.Error("Done() should return channel for cancellable context")
	}

	// 取消并验证
	cancel()
	select {
	case <-done:
		// 符合预期
	case <-time.After(100 * time.Millisecond):
		t.Error("Done() channel should be closed after cancel")
	}
}

// TestRouteContext_Err 测试 Err 方法
func TestRouteContext_Err(t *testing.T) {
	// 测试未取消的 context
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	err := ctx.Err()
	if err != nil {
		t.Errorf("Err() = %v; want nil", err)
	}

	// 测试已取消的 context
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()

	ctx2 := acquireContext(parentCtx, &paramPairs, 0)
	defer releaseContext(ctx2)

	err = ctx2.Err()
	if err != context.Canceled {
		t.Errorf("Err() = %v; want context.Canceled", err)
	}
}

// TestGetParam 测试全局 GetParam 辅助函数
func TestGetParam(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "456"}
	paramCount := 1

	ctx := acquireContext(parent, &paramPairs, paramCount)
	defer releaseContext(ctx)

	// 测试 routeContext
	val, ok := GetParam(ctx, "id")
	if !ok || val != "456" {
		t.Errorf("GetParam(id) = %v, %v; want 456, true", val, ok)
	}

	// 测试非 routeContext
	val, ok = GetParam(context.Background(), "id")
	if ok || val != "" {
		t.Errorf("GetParam on non-routeContext = %v, %v; want '', false", val, ok)
	}
}

// TestSetValue 测试全局 SetValue 辅助函数
func TestSetValue(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// 测试 routeContext
	ok := SetValue(ctx, "key", "value")
	if !ok {
		t.Error("SetValue should succeed on routeContext")
	}

	val, _ := GetParam(ctx, "key")
	_ = val

	// 测试非 routeContext
	ok = SetValue(context.Background(), "key", "value")
	if ok {
		t.Error("SetValue should fail on non-routeContext")
	}
}

// TestReleaseContext 测试全局 ReleaseContext 辅助函数
func TestReleaseContext(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "789"}
	ctx := acquireContext(parent, &paramPairs, 1)

	ctx.SetValue("key", "value")

	// 释放 context
	ReleaseContext(ctx)

	// 验证 context 已重置
	if ctx.Context != nil {
		t.Error("Context should be nil after release")
	}
	if ctx.paramCount != 0 {
		t.Error("paramCount should be 0 after release")
	}
	if ctx.valueCount != 0 {
		t.Error("valueCount should be 0 after release")
	}

	// 测试非 routeContext（不应 panic）
	ReleaseContext(context.Background())
}

// TestContextPool 测试 context 池化机制
func TestContextPool(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "pool_test"}

	// 多次获取和释放
	for i := 0; i < 100; i++ {
		ctx := acquireContext(parent, &paramPairs, 1)
		val, ok := ctx.GetParam("id")
		if !ok || val != "pool_test" {
			t.Errorf("Iteration %d: GetParam(id) = %v, %v; want pool_test, true", i, val, ok)
		}
		releaseContext(ctx)
	}
}

// TestRouteContext_MaxParams 测试 MaxParams 限制
func TestRouteContext_MaxParams(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair

	// 填充所有参数
	for i := 0; i < MaxParams; i++ {
		paramPairs[i] = paramPair{key: "key", value: "value"}
	}

	ctx := acquireContext(parent, &paramPairs, MaxParams)
	defer releaseContext(ctx)

	if ctx.paramCount != MaxParams {
		t.Errorf("paramCount = %d; want %d", ctx.paramCount, MaxParams)
	}
}

// BenchmarkRouteContext_GetParam 基准测试 GetParam 性能
func BenchmarkRouteContext_GetParam(b *testing.B) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "123"}
	paramPairs[1] = paramPair{key: "name", value: "test"}
	ctx := acquireContext(parent, &paramPairs, 2)
	defer releaseContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ctx.GetParam("name")
	}
}

// BenchmarkRouteContext_SetValue 基准测试 SetValue 性能
func BenchmarkRouteContext_SetValue(b *testing.B) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := acquireContext(parent, &paramPairs, 0)
		ctx.SetValue("key", "value")
		releaseContext(ctx)
	}
}

// BenchmarkRouteContext_Value 基准测试 Value 方法性能
func BenchmarkRouteContext_Value(b *testing.B) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	ctx.SetValue("key", "value")
	defer releaseContext(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctx.Value("key")
	}
}

// BenchmarkContextPool 基准测试 context 池化性能
func BenchmarkContextPool(b *testing.B) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "123"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := acquireContext(parent, &paramPairs, 1)
		releaseContext(ctx)
	}
}

// BenchmarkContextPool_Parallel 基准测试并行 context 池化性能
func BenchmarkContextPool_Parallel(b *testing.B) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "123"}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx := acquireContext(parent, &paramPairs, 1)
			releaseContext(ctx)
		}
	})
}

// TestExecuteHandler 测试 ExecuteHandler 函数
func TestExecuteHandler(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "123"}

	// 测试成功执行
	ctx := acquireContext(parent, &paramPairs, 1)
	called := false
	handler := func(ctx context.Context) error {
		called = true
		id, ok := GetParam(ctx, "id")
		if !ok || id != "123" {
			t.Errorf("GetParam failed: got %v, %v", id, ok)
		}
		return nil
	}

	err := ExecuteHandler(ctx, handler, nil)
	if err != nil {
		t.Errorf("ExecuteHandler returned error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}

	// 验证 context 已被释放（Context 应该是 nil）
	if ctx.Context != nil {
		t.Error("context was not released after ExecuteHandler")
	}
}

// TestExecuteHandler_WithMiddleware 测试带中间件的 ExecuteHandler
func TestExecuteHandler_WithMiddleware(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair

	ctx := acquireContext(parent, &paramPairs, 0)

	// 跟踪执行顺序
	var order []string

	handler := func(ctx context.Context) error {
		order = append(order, "handler")
		return nil
	}

	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context) error {
			order = append(order, "middleware1_before")
			err := next(ctx)
			order = append(order, "middleware1_after")
			return err
		}
	}

	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context) error {
			order = append(order, "middleware2_before")
			err := next(ctx)
			order = append(order, "middleware2_after")
			return err
		}
	}

	err := ExecuteHandler(ctx, handler, []Middleware{middleware1, middleware2})
	if err != nil {
		t.Errorf("ExecuteHandler returned error: %v", err)
	}

	// 验证执行顺序：middleware1 -> middleware2 -> handler -> middleware2 -> middleware1
	expectedOrder := []string{
		"middleware1_before",
		"middleware2_before",
		"handler",
		"middleware2_after",
		"middleware1_after",
	}

	if len(order) != len(expectedOrder) {
		t.Fatalf("execution order length mismatch: got %d, want %d", len(order), len(expectedOrder))
	}

	for i, step := range expectedOrder {
		if order[i] != step {
			t.Errorf("execution order[%d]: got %s, want %s", i, order[i], step)
		}
	}
}

// TestExecuteHandler_WithError 测试当 handler 返回错误时的 ExecuteHandler
func TestExecuteHandler_WithError(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair

	ctx := acquireContext(parent, &paramPairs, 0)

	expectedErr := fmt.Errorf("test error")
	handler := func(ctx context.Context) error {
		return expectedErr
	}

	err := ExecuteHandler(ctx, handler, nil)
	if err != expectedErr {
		t.Errorf("ExecuteHandler returned wrong error: got %v, want %v", err, expectedErr)
	}

	// 验证即使有错误 context 仍然被释放
	if ctx.Context != nil {
		t.Error("context was not released after ExecuteHandler with error")
	}
}
