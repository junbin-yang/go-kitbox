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

// TestRouteContext_GetParam tests the GetParam method
func TestRouteContext_GetParam(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "123"}
	paramPairs[1] = paramPair{key: "name", value: "test"}
	paramCount := 2

	ctx := acquireContext(parent, &paramPairs, paramCount)
	defer releaseContext(ctx)

	// Test existing param
	val, ok := ctx.GetParam("id")
	if !ok || val != "123" {
		t.Errorf("GetParam(id) = %v, %v; want 123, true", val, ok)
	}

	val, ok = ctx.GetParam("name")
	if !ok || val != "test" {
		t.Errorf("GetParam(name) = %v, %v; want test, true", val, ok)
	}

	// Test non-existing param
	val, ok = ctx.GetParam("nonexistent")
	if ok || val != "" {
		t.Errorf("GetParam(nonexistent) = %v, %v; want '', false", val, ok)
	}
}

// TestRouteContext_SetValue tests the SetValue method
func TestRouteContext_SetValue(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// Test setting values
	ok := ctx.SetValue("key1", "value1")
	if !ok {
		t.Error("SetValue(key1) failed")
	}

	ok = ctx.SetValue("key2", 42)
	if !ok {
		t.Error("SetValue(key2) failed")
	}

	// Test retrieving values
	val, ok := ctx.GetValue("key1")
	if !ok || val != "value1" {
		t.Errorf("GetValue(key1) = %v, %v; want value1, true", val, ok)
	}

	val, ok = ctx.GetValue("key2")
	if !ok || val != 42 {
		t.Errorf("GetValue(key2) = %v, %v; want 42, true", val, ok)
	}

	// Test non-existing value
	val, ok = ctx.GetValue("nonexistent")
	if ok || val != nil {
		t.Errorf("GetValue(nonexistent) = %v, %v; want nil, false", val, ok)
	}
}

// TestRouteContext_SetValue_MaxValues tests the MaxValues limit
func TestRouteContext_SetValue_MaxValues(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// Fill up to MaxValues
	for i := 0; i < MaxValues; i++ {
		ok := ctx.SetValue("key", i)
		if !ok {
			t.Errorf("SetValue failed at index %d", i)
		}
	}

	// Try to exceed MaxValues
	ok := ctx.SetValue("overflow", "value")
	if ok {
		t.Error("SetValue should fail when MaxValues is exceeded")
	}
}

// TestRouteContext_Value tests the Value method (context.Context interface)
func TestRouteContext_Value(t *testing.T) {
	parent := context.WithValue(context.Background(), testParentKey, "parent_value")
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// Set a value in routeContext
	ctx.SetValue(testRouteKey, "route_value")

	// Test retrieving routeContext value via Value method
	val := ctx.Value(testRouteKey)
	if val != "route_value" {
		t.Errorf("Value(route_key) = %v; want route_value", val)
	}

	// Test retrieving parent context value
	val = ctx.Value(testParentKey)
	if val != "parent_value" {
		t.Errorf("Value(parent_key) = %v; want parent_value", val)
	}

	// Test non-existing value
	val = ctx.Value(testContextKey("nonexistent"))
	if val != nil {
		t.Errorf("Value(nonexistent) = %v; want nil", val)
	}

	// Test non-string key
	val = ctx.Value(123)
	if val != nil {
		t.Errorf("Value(123) = %v; want nil", val)
	}
}

// TestRouteContext_Deadline tests the Deadline method
func TestRouteContext_Deadline(t *testing.T) {
	// Test with parent that has no deadline
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	deadline, ok := ctx.Deadline()
	if ok {
		t.Errorf("Deadline() = %v, %v; want zero, false", deadline, ok)
	}

	// Test with parent that has deadline
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

// TestRouteContext_Done tests the Done method
func TestRouteContext_Done(t *testing.T) {
	// Test with parent that is not cancellable
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	done := ctx.Done()
	if done != nil {
		t.Error("Done() should return nil for non-cancellable context")
	}

	// Test with cancellable parent
	parentCtx, cancel := context.WithCancel(context.Background())
	ctx2 := acquireContext(parentCtx, &paramPairs, 0)
	defer releaseContext(ctx2)

	done = ctx2.Done()
	if done == nil {
		t.Error("Done() should return channel for cancellable context")
	}

	// Cancel and verify
	cancel()
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Done() channel should be closed after cancel")
	}
}

// TestRouteContext_Err tests the Err method
func TestRouteContext_Err(t *testing.T) {
	// Test with non-cancelled context
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	err := ctx.Err()
	if err != nil {
		t.Errorf("Err() = %v; want nil", err)
	}

	// Test with cancelled context
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()

	ctx2 := acquireContext(parentCtx, &paramPairs, 0)
	defer releaseContext(ctx2)

	err = ctx2.Err()
	if err != context.Canceled {
		t.Errorf("Err() = %v; want context.Canceled", err)
	}
}

// TestGetParam tests the global GetParam helper function
func TestGetParam(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "456"}
	paramCount := 1

	ctx := acquireContext(parent, &paramPairs, paramCount)
	defer releaseContext(ctx)

	// Test with routeContext
	val, ok := GetParam(ctx, "id")
	if !ok || val != "456" {
		t.Errorf("GetParam(id) = %v, %v; want 456, true", val, ok)
	}

	// Test with non-routeContext
	val, ok = GetParam(context.Background(), "id")
	if ok || val != "" {
		t.Errorf("GetParam on non-routeContext = %v, %v; want '', false", val, ok)
	}
}

// TestSetValue tests the global SetValue helper function
func TestSetValue(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	ctx := acquireContext(parent, &paramPairs, 0)
	defer releaseContext(ctx)

	// Test with routeContext
	ok := SetValue(ctx, "key", "value")
	if !ok {
		t.Error("SetValue should succeed on routeContext")
	}

	val, _ := GetParam(ctx, "key")
	_ = val

	// Test with non-routeContext
	ok = SetValue(context.Background(), "key", "value")
	if ok {
		t.Error("SetValue should fail on non-routeContext")
	}
}

// TestReleaseContext tests the global ReleaseContext helper function
func TestReleaseContext(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "789"}
	ctx := acquireContext(parent, &paramPairs, 1)

	ctx.SetValue("key", "value")

	// Release context
	ReleaseContext(ctx)

	// Verify context is reset
	if ctx.Context != nil {
		t.Error("Context should be nil after release")
	}
	if ctx.paramCount != 0 {
		t.Error("paramCount should be 0 after release")
	}
	if ctx.valueCount != 0 {
		t.Error("valueCount should be 0 after release")
	}

	// Test with non-routeContext (should not panic)
	ReleaseContext(context.Background())
}

// TestContextPool tests the context pooling mechanism
func TestContextPool(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "pool_test"}

	// Acquire and release multiple times
	for i := 0; i < 100; i++ {
		ctx := acquireContext(parent, &paramPairs, 1)
		val, ok := ctx.GetParam("id")
		if !ok || val != "pool_test" {
			t.Errorf("Iteration %d: GetParam(id) = %v, %v; want pool_test, true", i, val, ok)
		}
		releaseContext(ctx)
	}
}

// TestRouteContext_MaxParams tests the MaxParams limit
func TestRouteContext_MaxParams(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair

	// Fill all params
	for i := 0; i < MaxParams; i++ {
		paramPairs[i] = paramPair{key: "key", value: "value"}
	}

	ctx := acquireContext(parent, &paramPairs, MaxParams)
	defer releaseContext(ctx)

	if ctx.paramCount != MaxParams {
		t.Errorf("paramCount = %d; want %d", ctx.paramCount, MaxParams)
	}
}

// BenchmarkRouteContext_GetParam benchmarks GetParam performance
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

// BenchmarkRouteContext_SetValue benchmarks SetValue performance
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

// BenchmarkRouteContext_Value benchmarks Value method performance
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

// BenchmarkContextPool benchmarks context pooling
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

// BenchmarkContextPool_Parallel benchmarks parallel context pooling
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

// TestExecuteHandler tests the ExecuteHandler function
func TestExecuteHandler(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair
	paramPairs[0] = paramPair{key: "id", value: "123"}

	// Test successful execution
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

	// Verify context was released (Context should be nil)
	if ctx.Context != nil {
		t.Error("context was not released after ExecuteHandler")
	}
}

// TestExecuteHandler_WithMiddleware tests ExecuteHandler with middleware
func TestExecuteHandler_WithMiddleware(t *testing.T) {
	parent := context.Background()
	var paramPairs [MaxParams]paramPair

	ctx := acquireContext(parent, &paramPairs, 0)

	// Track execution order
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

	// Verify execution order: middleware1 -> middleware2 -> handler -> middleware2 -> middleware1
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

// TestExecuteHandler_WithError tests ExecuteHandler when handler returns error
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

	// Verify context was still released even with error
	if ctx.Context != nil {
		t.Error("context was not released after ExecuteHandler with error")
	}
}
