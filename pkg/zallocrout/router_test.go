package zallocrout

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// 测试创建路由器
func TestNewRouter(t *testing.T) {
	router := NewRouter()
	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
	if router.roots == nil {
		t.Error("router.roots is nil")
	}
	if router.hotCache == nil {
		t.Error("router.hotCache is nil")
	}
	if router.metrics == nil {
		t.Error("router.metrics is nil")
	}
}

// 测试静态路由
func TestRouter_StaticRoute(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	err := router.AddRoute("GET", "/api/users", handler)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	// 测试匹配成功
	ctx, h, _, ok := router.Match("GET", "/api/users", context.Background())
	if !ok {
		t.Fatal("Match failed")
	}
	defer ReleaseContext(ctx)

	if h == nil {
		t.Error("handler is nil")
	}

	// 测试匹配失败
	_, _, _, ok = router.Match("GET", "/api/posts", context.Background())
	if ok {
		t.Error("Match should fail for non-existent route")
	}
}

// 测试参数路由
func TestRouter_ParamRoute(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	err := router.AddRoute("GET", "/users/:id", handler)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	// 测试匹配成功
	ctx, _, _, ok := router.Match("GET", "/users/123", context.Background())
	if !ok {
		t.Fatal("Match failed")
	}
	defer ReleaseContext(ctx)

	// 验证参数提取
	id, ok := GetParam(ctx, "id")
	if !ok || id != "123" {
		t.Errorf("param id = %v, want 123", id)
	}
}

// 测试通配符路由
func TestRouter_WildcardRoute(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	err := router.AddRoute("GET", "/files/*path", handler)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	// 测试匹配成功
	ctx, _, _, ok := router.Match("GET", "/files/docs/readme.md", context.Background())
	if !ok {
		t.Fatal("Match failed")
	}
	defer ReleaseContext(ctx)

	// 验证通配符参数
	wildcard, ok := GetParam(ctx, "*")
	if !ok || wildcard != "docs/readme.md" {
		t.Errorf("param * = %v, want docs/readme.md", wildcard)
	}
}

// 测试复杂路由
func TestRouter_ComplexRoute(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }

	// 注册多个路由
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/users"},
		{"GET", "/api/v1/users/:id"},
		{"POST", "/api/v1/users"},
		{"GET", "/api/v1/users/:id/posts"},
		{"GET", "/api/v1/users/:id/posts/:postId"},
		{"GET", "/files/*path"},
	}

	for _, route := range routes {
		err := router.AddRoute(route.method, route.path, handler)
		if err != nil {
			t.Fatalf("AddRoute(%s, %s) failed: %v", route.method, route.path, err)
		}
	}

	// 测试匹配
	tests := []struct {
		method         string
		path           string
		shouldMatch    bool
		expectedParams map[string]string
	}{
		{"GET", "/api/v1/users", true, map[string]string{}},
		{"GET", "/api/v1/users/123", true, map[string]string{"id": "123"}},
		{"POST", "/api/v1/users", true, map[string]string{}},
		{"GET", "/api/v1/users/123/posts", true, map[string]string{"id": "123"}},
		{"GET", "/api/v1/users/123/posts/456", true, map[string]string{"id": "123", "postId": "456"}},
		{"GET", "/files/docs/readme.md", true, map[string]string{"*": "docs/readme.md"}},
		{"GET", "/api/v1/posts", false, nil},
		{"DELETE", "/api/v1/users", false, nil},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.path), func(t *testing.T) {
			ctx, _, _, ok := router.Match(tt.method, tt.path, context.Background())
			if ok != tt.shouldMatch {
				t.Errorf("Match(%s, %s) = %v, want %v", tt.method, tt.path, ok, tt.shouldMatch)
				return
			}

			if ok {
				defer ReleaseContext(ctx)
				for k, v := range tt.expectedParams {
					val, found := GetParam(ctx, k)
					if !found || val != v {
						t.Errorf("param %s = %v, want %v", k, val, v)
					}
				}
			}
		})
	}
}

// TestRouter_ParamNameConflict tests the bug where overlapping routes
// with different param names cause the first param name to be used
func TestRouter_ParamNameConflict(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }

	// Register routes in the same order as HTTP example
	_ = router.AddRoute("GET", "/users/:id", handler)
	_ = router.AddRoute("GET", "/users/:userId/posts/:postId", handler)

	// Test single param route
	ctx1, _, _, ok := router.Match("GET", "/users/123", context.Background())
	if !ok {
		t.Fatal("Match /users/123 failed")
	}
	defer ReleaseContext(ctx1)

	id1, ok := GetParam(ctx1, "id")
	if !ok || id1 != "123" {
		t.Errorf("GetParam(id) = %v, %v; want 123, true", id1, ok)
	}

	// Test multi-param route
	ctx2, _, _, ok := router.Match("GET", "/users/123/posts/456", context.Background())
	if !ok {
		t.Fatal("Match /users/123/posts/456 failed")
	}
	defer ReleaseContext(ctx2)

	// This should work with the fix
	userId, ok := GetParam(ctx2, "userId")
	if !ok || userId != "123" {
		t.Errorf("GetParam(userId) = %v, %v; want 123, true", userId, ok)
	}

	postId, ok := GetParam(ctx2, "postId")
	if !ok || postId != "456" {
		t.Errorf("GetParam(postId) = %v, %v; want 456, true", postId, ok)
	}
}

// 测试路由优先级（静态 > 参数 > 通配符）
func TestRouter_RoutePriority(t *testing.T) {
	router := NewRouter()

	staticHandler := func(ctx context.Context) error { return nil }
	paramHandler := func(ctx context.Context) error { return nil }

	// 注册路由
	_ = router.AddRoute("GET", "/users/admin", staticHandler)
	_ = router.AddRoute("GET", "/users/:id", paramHandler)

	// 匹配 /users/admin 应该匹配静态路由
	ctx, _, _, ok := router.Match("GET", "/users/admin", context.Background())
	if !ok {
		t.Fatal("Match failed")
	}
	defer ReleaseContext(ctx)

	// 验证匹配的是静态路由（没有参数）
	rctx := ctx.(*routeContext)
	if rctx.paramCount != 0 {
		t.Error("should match static route, not param route")
	}

	// 匹配 /users/123 应该匹配参数路由
	ctx2, _, _, ok := router.Match("GET", "/users/123", context.Background())
	if !ok {
		t.Fatal("Match failed")
	}
	defer ReleaseContext(ctx2)

	id, ok := GetParam(ctx2, "id")
	if !ok || id != "123" {
		t.Errorf("param id = %v, want 123", id)
	}
}

// 测试中间件
func TestRouter_Middlewares(t *testing.T) {
	router := NewRouter()

	var order []string
	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context) error {
			order = append(order, "m1")
			return next(ctx)
		}
	}
	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context) error {
			order = append(order, "m2")
			return next(ctx)
		}
	}

	handler := func(ctx context.Context) error {
		order = append(order, "handler")
		return nil
	}

	_ = router.AddRoute("GET", "/test", handler, middleware1, middleware2)

	ctx, h, mws, ok := router.Match("GET", "/test", context.Background())
	if !ok {
		t.Fatal("Match failed")
	}

	// 执行处理器链
	err := ExecuteHandler(ctx, h, mws)
	if err != nil {
		t.Fatalf("ExecuteHandler failed: %v", err)
	}

	expectedOrder := []string{"m1", "m2", "handler"}
	if len(order) != len(expectedOrder) {
		t.Fatalf("execution order length = %d, want %d", len(order), len(expectedOrder))
	}
	for i, expected := range expectedOrder {
		if order[i] != expected {
			t.Errorf("order[%d] = %s, want %s", i, order[i], expected)
		}
	}
}

// 测试热点缓存
func TestRouter_HotCache(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/users/:id", handler)

	// 匹配多次以触发缓存
	for i := 0; i < hotCacheThreshold+10; i++ {
		ctx, _, _, ok := router.Match("GET", "/users/123", context.Background())
		if !ok {
			t.Fatalf("Match %d failed", i)
		}
		ReleaseContext(ctx)
	}

	// 验证缓存命中
	metrics := router.Metrics()
	defer router.ResetMetrics()
	if metrics.CacheHits == 0 {
		t.Error("cache hits should be > 0")
	}

	hitRate := router.CacheHitRate()
	if hitRate == 0 {
		t.Error("cache hit rate should be > 0")
	}
}

// 测试禁用热点缓存
func TestRouter_DisableHotCache(t *testing.T) {
	router := NewRouter()
	router.DisableHotCache()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/users/:id", handler)

	// 匹配多次
	for i := 0; i < hotCacheThreshold+10; i++ {
		ctx, _, _, ok := router.Match("GET", "/users/123", context.Background())
		if !ok {
			t.Fatalf("Match %d failed", i)
		}
		ReleaseContext(ctx)
	}

	// 验证没有缓存命中
	metrics := router.Metrics()
	if metrics.CacheHits != 0 {
		t.Errorf("cache hits = %d, want 0", metrics.CacheHits)
	}
}

// 测试清空热点缓存
func TestRouter_ClearHotCache(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/users/:id", handler)

	// 触发缓存
	for i := 0; i < hotCacheThreshold+10; i++ {
		ctx, _, _, ok := router.Match("GET", "/users/123", context.Background())
		if !ok {
			t.Fatalf("Match %d failed", i)
		}
		ReleaseContext(ctx)
	}

	// 清空缓存
	router.ClearHotCache()

	// 验证缓存已清空
	totalEntries, _ := router.CacheStats()
	if totalEntries != 0 {
		t.Errorf("cache entries = %d, want 0", totalEntries)
	}
}

// 测试性能指标
func TestRouter_Metrics(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/static", handler)
	_ = router.AddRoute("GET", "/users/:id", handler)
	_ = router.AddRoute("GET", "/files/*path", handler)

	metrics := router.Metrics()
	if metrics.StaticRoutes != 1 {
		t.Errorf("static routes = %d, want 1", metrics.StaticRoutes)
	}
	if metrics.ParamRoutes != 1 {
		t.Errorf("param routes = %d, want 1", metrics.ParamRoutes)
	}
	if metrics.WildcardRoutes != 1 {
		t.Errorf("wildcard routes = %d, want 1", metrics.WildcardRoutes)
	}
}

// 测试启用热点缓存
func TestRouter_EnableHotCache(t *testing.T) {
	router := NewRouter()
	router.DisableHotCache()
	router.EnableHotCache()

	if atomic.LoadUint32(&router.enableHotCache) != 1 {
		t.Error("hot cache should be enabled")
	}
}

// 测试指标重置
func TestRouter_MetricsReset(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/test", handler)

	// 触发一些匹配
	for i := 0; i < 10; i++ {
		ctx, _, _, _ := router.Match("GET", "/test", context.Background())
		ReleaseContext(ctx)
	}

	metrics := router.Metrics()
	if metrics.TotalMatches != 10 {
		t.Errorf("total matches = %d, want 10", metrics.TotalMatches)
	}
}

// 测试非法路由
func TestRouter_InvalidRoute(t *testing.T) {
	router := NewRouter()
	handler := func(ctx context.Context) error { return nil }

	tests := []struct {
		name string
		path string
	}{
		{"空路径", ""},
		{"无开头斜杠", "api/users"},
		{"双斜杠", "//api//users"},
		{"点斜杠", "//api/./users"},
		{"父目录", "//api/../users"},
		{"空参数", "/users/:"},
		{"空通配符", "/files/*"},
		{"通配符不在末尾", "/files/*/list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.AddRoute("GET", tt.path, handler)
			if err == nil {
				t.Errorf("AddRoute(%s) should fail", tt.path)
			}
		})
	}
}

// 测试并发访问
func TestRouter_ConcurrentAccess(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/users/:id", handler)

	var wg sync.WaitGroup
	goroutines := 100
	iterations := 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ctx, _, _, ok := router.Match("GET", fmt.Sprintf("/users/%d", j), context.Background())
				if !ok {
					t.Errorf("Match failed")
					return
				}
				ReleaseContext(ctx)
			}
		}()
	}

	wg.Wait()
}

// 测试缓存命中率为零的情况
func TestRouter_CacheHitRateZero(t *testing.T) {
	router := NewRouter()
	hitRate := router.CacheHitRate()
	if hitRate != 0 {
		t.Errorf("cache hit rate = %f, want 0", hitRate)
	}
}

// 测试路径规范化
func TestRouter_PathNormalization(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/api/users", handler)

	// 测试需要规范化的路径
	tests := []string{
		"/api/users/",      // 结尾斜杠
		"/api//users",      // 双斜杠
		"/api/./users",     // 当前目录
		"/api/v1/../users", // 父目录
	}

	for _, path := range tests {
		ctx, _, _, ok := router.Match("GET", path, context.Background())
		if !ok {
			t.Errorf("Match(%s) failed", path)
			continue
		}
		ReleaseContext(ctx)
	}
}

// 测试静态路由和参数路由优先级冲突
func TestRouter_StaticVsParamPriority(t *testing.T) {
	router := NewRouter()

	staticCalled := false
	paramCalled := false

	staticHandler := func(ctx context.Context) error {
		staticCalled = true
		return nil
	}
	paramHandler := func(ctx context.Context) error {
		paramCalled = true
		return nil
	}

	// 注册路由（注册顺序不应影响优先级）
	_ = router.AddRoute("GET", "/api/:resource", paramHandler)
	_ = router.AddRoute("GET", "/api/users", staticHandler)

	// 测试访问静态路由（应该匹配静态路由，不是参数路由）
	ctx, handler, _, ok := router.Match("GET", "/api/users", context.Background())
	if !ok {
		t.Fatal("Match failed")
	}
	defer ReleaseContext(ctx)

	// 执行处理器
	_ = handler(ctx)

	if !staticCalled {
		t.Error("Should call static handler")
	}
	if paramCalled {
		t.Error("Should not call param handler")
	}

	// 验证没有参数（确认匹配的是静态路由）
	if _, ok := GetParam(ctx, "resource"); ok {
		t.Error("Static route should not have parameters")
	}
}

// 测试参数路由和通配符路由优先级
func TestRouter_ParamVsWildcardPriority(t *testing.T) {
	router := NewRouter()

	paramHandler := func(ctx context.Context) error { return nil }
	wildcardHandler := func(ctx context.Context) error { return nil }

	// 注册路由（使用不同的路径前缀避免冲突）
	_ = router.AddRoute("GET", "/api/:resource", paramHandler)
	_ = router.AddRoute("GET", "/files/*path", wildcardHandler)

	// 测试参数路由
	ctx1, _, _, ok := router.Match("GET", "/api/users", context.Background())
	if !ok {
		t.Fatal("Match /api/users failed")
	}
	defer ReleaseContext(ctx1)

	resource, ok := GetParam(ctx1, "resource")
	if !ok || resource != "users" {
		t.Errorf("param resource = %v, want users", resource)
	}

	// 测试通配符路由（匹配剩余所有路径）
	ctx2, _, _, ok := router.Match("GET", "/files/docs/readme.txt", context.Background())
	if !ok {
		t.Fatal("Match /files/docs/readme.txt failed")
	}
	defer ReleaseContext(ctx2)

	path, ok := GetParam(ctx2, "*")
	if !ok || path != "docs/readme.txt" {
		t.Errorf("param * = %v, want docs/readme.txt", path)
	}

	// 测试通配符路由匹配单个文件
	ctx3, _, _, ok := router.Match("GET", "/files/readme.txt", context.Background())
	if !ok {
		t.Fatal("Match /files/readme.txt failed")
	}
	defer ReleaseContext(ctx3)

	path2, ok := GetParam(ctx3, "*")
	if !ok || path2 != "readme.txt" {
		t.Errorf("param * = %v, want readme.txt", path2)
	}
}

// 测试多个参数路由的复杂匹配
func TestRouter_MultipleParamRoutes(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }

	// 注册多个重叠的参数路由
	routes := []string{
		"/users/:id",
		"/users/:userId/posts",
		"/users/:userId/posts/:postId",
		"/users/:userId/posts/:postId/comments",
		"/users/:userId/posts/:postId/comments/:commentId",
	}

	for _, route := range routes {
		err := router.AddRoute("GET", route, handler)
		if err != nil {
			t.Fatalf("AddRoute(%s) failed: %v", route, err)
		}
	}

	// 测试不同深度的路径匹配
	tests := []struct {
		path           string
		expectedParams map[string]string
	}{
		{"/users/123", map[string]string{"id": "123"}},
		{"/users/123/posts", map[string]string{"userId": "123"}},
		{"/users/123/posts/456", map[string]string{"userId": "123", "postId": "456"}},
		{"/users/123/posts/456/comments", map[string]string{"userId": "123", "postId": "456"}},
		{"/users/123/posts/456/comments/789", map[string]string{"userId": "123", "postId": "456", "commentId": "789"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ctx, _, _, ok := router.Match("GET", tt.path, context.Background())
			if !ok {
				t.Fatalf("Match(%s) failed", tt.path)
			}
			defer ReleaseContext(ctx)

			for key, expectedVal := range tt.expectedParams {
				val, ok := GetParam(ctx, key)
				if !ok || val != expectedVal {
					t.Errorf("param %s = %v, want %v", key, val, expectedVal)
				}
			}
		})
	}
}

// 测试静态、参数、通配符路由混合场景
func TestRouter_MixedRouteTypes(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }

	// 注册混合路由
	_ = router.AddRoute("GET", "/static", handler)
	_ = router.AddRoute("GET", "/static/page", handler)
	_ = router.AddRoute("GET", "/dynamic/:id", handler)
	_ = router.AddRoute("GET", "/dynamic/:id/edit", handler)
	_ = router.AddRoute("GET", "/files/*path", handler)

	tests := []struct {
		path           string
		shouldMatch    bool
		expectedParams map[string]string
	}{
		{"/static", true, map[string]string{}},
		{"/static/page", true, map[string]string{}},
		{"/static/other", false, nil},
		{"/dynamic/123", true, map[string]string{"id": "123"}},
		{"/dynamic/123/edit", true, map[string]string{"id": "123"}},
		{"/dynamic/123/delete", false, nil},
		{"/files/a/b/c/d.txt", true, map[string]string{"*": "a/b/c/d.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ctx, _, _, ok := router.Match("GET", tt.path, context.Background())
			if ok != tt.shouldMatch {
				t.Errorf("Match(%s) = %v, want %v", tt.path, ok, tt.shouldMatch)
				return
			}

			if ok {
				defer ReleaseContext(ctx)
				for key, expectedVal := range tt.expectedParams {
					val, found := GetParam(ctx, key)
					if !found || val != expectedVal {
						t.Errorf("param %s = %v, want %v", key, val, expectedVal)
					}
				}
			}
		})
	}
}

// 测试同名参数在不同位置
func TestRouter_SameParamNameDifferentPosition(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }

	// 注册具有相同参数名但在不同位置的路由
	_ = router.AddRoute("GET", "/api/:id", handler)
	_ = router.AddRoute("GET", "/users/:id/profile", handler)
	_ = router.AddRoute("GET", "/posts/:id/author/:id", handler) // 同名参数出现两次

	// 测试 /api/:id
	ctx1, _, _, ok := router.Match("GET", "/api/123", context.Background())
	if !ok {
		t.Fatal("Match /api/123 failed")
	}
	defer ReleaseContext(ctx1)

	id1, ok := GetParam(ctx1, "id")
	if !ok || id1 != "123" {
		t.Errorf("GetParam(id) = %v, want 123", id1)
	}

	// 测试 /users/:id/profile
	ctx2, _, _, ok := router.Match("GET", "/users/456/profile", context.Background())
	if !ok {
		t.Fatal("Match /users/456/profile failed")
	}
	defer ReleaseContext(ctx2)

	id2, ok := GetParam(ctx2, "id")
	if !ok || id2 != "456" {
		t.Errorf("GetParam(id) = %v, want 456", id2)
	}
}

// 测试路由路径边界情况
func TestRouter_EdgeCasePaths(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }

	// 注册根路径
	_ = router.AddRoute("GET", "/", handler)

	// 测试根路径
	ctx, _, _, ok := router.Match("GET", "/", context.Background())
	if !ok {
		t.Error("Should match root path /")
	} else {
		ReleaseContext(ctx)
	}

	// 测试带结尾斜杠的根路径（规范化后应该匹配）
	ctx2, _, _, ok := router.Match("GET", "//", context.Background())
	if !ok {
		t.Error("Should match normalized root path //")
	} else {
		ReleaseContext(ctx2)
	}
}

// 测试特殊字符在参数中
func TestRouter_SpecialCharsInParams(t *testing.T) {
	router := NewRouter()

	handler := func(ctx context.Context) error { return nil }
	_ = router.AddRoute("GET", "/users/:id", handler)

	tests := []struct {
		path     string
		expected string
	}{
		{"/users/user-123", "user-123"},
		{"/users/user_456", "user_456"},
		{"/users/user.789", "user.789"},
		{"/users/123abc", "123abc"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ctx, _, _, ok := router.Match("GET", tt.path, context.Background())
			if !ok {
				t.Fatalf("Match(%s) failed", tt.path)
			}
			defer ReleaseContext(ctx)

			id, ok := GetParam(ctx, "id")
			if !ok || id != tt.expected {
				t.Errorf("GetParam(id) = %v, want %v", id, tt.expected)
			}
		})
	}
}

// 测试不同HTTP方法的路由隔离
func TestRouter_MethodIsolation(t *testing.T) {
	router := NewRouter()

	getHandler := func(ctx context.Context) error { return nil }
	postHandler := func(ctx context.Context) error { return nil }

	// 同一路径注册不同方法
	_ = router.AddRoute("GET", "/api/users", getHandler)
	_ = router.AddRoute("POST", "/api/users", postHandler)

	// 测试 GET 方法
	ctx1, h1, _, ok := router.Match("GET", "/api/users", context.Background())
	if !ok {
		t.Error("GET /api/users should match")
	} else {
		ReleaseContext(ctx1)
		if fmt.Sprintf("%p", h1) != fmt.Sprintf("%p", getHandler) {
			t.Error("Should match GET handler")
		}
	}

	// 测试 POST 方法
	ctx2, h2, _, ok := router.Match("POST", "/api/users", context.Background())
	if !ok {
		t.Error("POST /api/users should match")
	} else {
		ReleaseContext(ctx2)
		if fmt.Sprintf("%p", h2) != fmt.Sprintf("%p", postHandler) {
			t.Error("Should match POST handler")
		}
	}

	// 测试未注册的方法
	_, _, _, ok = router.Match("DELETE", "/api/users", context.Background())
	if ok {
		t.Error("DELETE /api/users should not match")
	}
}
