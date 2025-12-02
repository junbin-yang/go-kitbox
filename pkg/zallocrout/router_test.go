package zallocrout

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// 测试路由器创建
func TestNewRouter(t *testing.T) {
	router := NewRouter()
	if router == nil {
		t.Fatal("NewRouter returned nil")
	}
	if router.roots == nil {
		t.Error("router roots not initialized")
	}
	if router.hotCache == nil {
		t.Error("router hotCache not initialized")
	}
	if router.metrics == nil {
		t.Error("router metrics not initialized")
	}
}

// 测试静态路由注册和匹配
func TestRouter_StaticRoute(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册路由
	err := router.AddRoute("GET", "/api/v1/users", handler)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	// 匹配路由
	result, ok := router.Match("GET", "/api/v1/users")
	if !ok {
		t.Fatal("Match failed")
	}
	defer result.Release()

	if result.Handler == nil {
		t.Error("handler is nil")
	}
	if result.paramCount != 0 {
		t.Errorf("params count = %d, want 0", result.paramCount)
	}
}

// 测试参数路由注册和匹配
func TestRouter_ParamRoute(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册路由
	err := router.AddRoute("GET", "/users/:id", handler)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	// 匹配路由
	result, ok := router.Match("GET", "/users/123")
	if !ok {
		t.Fatal("Match failed")
	}
	defer result.Release()

	id, ok := result.GetParam("id")
	if !ok || id != "123" {
		t.Errorf("param id = %v, want 123", id)
	}
}

// 测试通配符路由注册和匹配
func TestRouter_WildcardRoute(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册路由
	err := router.AddRoute("GET", "/files/*path", handler)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	// 匹配路由
	result, ok := router.Match("GET", "/files/docs/readme.md")
	if !ok {
		t.Fatal("Match failed")
	}
	defer result.Release()

	wildcard, ok := result.GetParam("*")
	if !ok || wildcard != "docs/readme.md" {
		t.Errorf("param * = %v, want docs/readme.md", wildcard)
	}
}

// 测试复杂路由
func TestRouter_ComplexRoute(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

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
		method       string
		path         string
		shouldMatch  bool
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
			result, ok := router.Match(tt.method, tt.path)
			if ok != tt.shouldMatch {
				t.Errorf("Match(%s, %s) = %v, want %v", tt.method, tt.path, ok, tt.shouldMatch)
				return
			}

			if ok {
				defer result.Release()
				for k, v := range tt.expectedParams {
					val, found := result.GetParam(k)
					if !found || val != v {
						t.Errorf("param %s = %v, want %v", k, val, v)
					}
				}
			}
		})
	}
}

// 测试路由优先级（静态 > 参数 > 通配符）
func TestRouter_RoutePriority(t *testing.T) {
	router := NewRouter()

	staticHandler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	paramHandler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册路由
	router.AddRoute("GET", "/users/admin", staticHandler)
	router.AddRoute("GET", "/users/:id", paramHandler)

	// 匹配 /users/admin 应该匹配静态路由
	result, ok := router.Match("GET", "/users/admin")
	if !ok {
		t.Fatal("Match failed")
	}
	defer result.Release()

	// 验证匹配的是静态路由（没有参数）
	if result.paramCount != 0 {
		t.Error("should match static route, not param route")
	}

	// 匹配 /users/123 应该匹配参数路由
	result2, ok := router.Match("GET", "/users/123")
	if !ok {
		t.Fatal("Match failed")
	}
	defer result2.Release()

	id, ok := result2.GetParam("id")
	if !ok || id != "123" {
		t.Error("should match param route")
	}
}

// 测试中间件
func TestRouter_Middlewares(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	middleware1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, params map[string]string) {
			next(w, r, params)
		}
	}
	middleware2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, params map[string]string) {
			next(w, r, params)
		}
	}

	// 注册带中间件的路由
	err := router.AddRoute("GET", "/api/users", handler, middleware1, middleware2)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	// 匹配路由
	result, ok := router.Match("GET", "/api/users")
	if !ok {
		t.Fatal("Match failed")
	}
	defer result.Release()

	if len(result.Middlewares) != 2 {
		t.Errorf("middlewares length = %d, want 2", len(result.Middlewares))
	}
}

// 测试热点缓存
func TestRouter_HotCache(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册路由
	router.AddRoute("GET", "/users/:id", handler)

	// 多次匹配同一路由（触发热点缓存）
	for i := 0; i < hotCacheThreshold+10; i++ {
		result, ok := router.Match("GET", "/users/123")
		if !ok {
			t.Fatal("Match failed")
		}
		result.Release()
	}

	// 检查缓存命中率
	hitRate := router.CacheHitRate()
	if hitRate == 0 {
		t.Error("cache hit rate should be > 0")
	}

	// 检查缓存统计
	totalEntries, _ := router.CacheStats()
	if totalEntries == 0 {
		t.Error("cache should have entries")
	}
}

// 测试禁用热点缓存
func TestRouter_DisableHotCache(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册路由
	router.AddRoute("GET", "/users/:id", handler)

	// 禁用热点缓存
	router.DisableHotCache()

	// 多次匹配
	for i := 0; i < hotCacheThreshold+10; i++ {
		result, ok := router.Match("GET", "/users/123")
		if !ok {
			t.Fatal("Match failed")
		}
		result.Release()
	}

	// 缓存命中次数应该为 0
	metrics := router.Metrics()
	if metrics.CacheHits != 0 {
		t.Errorf("cache hits = %d, want 0", metrics.CacheHits)
	}
}

// 测试清空热点缓存
func TestRouter_ClearHotCache(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册路由
	router.AddRoute("GET", "/users/:id", handler)

	// 触发热点缓存
	for i := 0; i < hotCacheThreshold+10; i++ {
		result, ok := router.Match("GET", "/users/123")
		if !ok {
			t.Fatal("Match failed")
		}
		result.Release()
	}

	// 清空缓存
	router.ClearHotCache()

	// 检查缓存统计
	totalEntries, _ := router.CacheStats()
	if totalEntries != 0 {
		t.Errorf("cache entries = %d, want 0", totalEntries)
	}
}

// 测试性能指标
func TestRouter_Metrics(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	// 注册不同类型的路由
	router.AddRoute("GET", "/api/users", handler)
	router.AddRoute("GET", "/users/:id", handler)
	router.AddRoute("GET", "/files/*path", handler)

	// 匹配路由
	router.Match("GET", "/api/users")
	router.Match("GET", "/users/123")
	router.Match("GET", "/files/docs/readme.md")

	// 检查指标
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
	if metrics.TotalMatches != 3 {
		t.Errorf("total matches = %d, want 3", metrics.TotalMatches)
	}
}

// 测试启用热点缓存
func TestRouter_EnableHotCache(t *testing.T) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	router.AddRoute("GET", "/users/:id", handler)

	// 禁用后再启用
	router.DisableHotCache()
	router.EnableHotCache()

	// 触发热点缓存
	for i := 0; i < hotCacheThreshold+10; i++ {
		result, _ := router.Match("GET", "/users/123")
		result.Release()
	}

	// 检查缓存命中
	hitRate := router.CacheHitRate()
	if hitRate == 0 {
		t.Error("cache should be enabled and have hits")
	}
}

// 测试指标重置
func TestRouter_MetricsReset(t *testing.T) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	router.AddRoute("GET", "/users", handler)

	// 匹配几次
	for i := 0; i < 5; i++ {
		router.Match("GET", "/users")
	}

	// 重置指标
	router.metrics.Reset()

	// 检查指标已重置
	metrics := router.Metrics()
	if metrics.TotalMatches != 0 {
		t.Errorf("total matches = %d, want 0 after reset", metrics.TotalMatches)
	}
}

// 测试非法路由
func TestRouter_InvalidRoute(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	tests := []struct {
		path    string
		wantErr bool
	}{
		{"", true},
		{"api/users", true},
		{"/api//users", true},
		{"/api/./users", true},
		{"/api/../users", true},
		{"/users/:", true},
		{"/files/*", true},
		{"/files/*/list", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := router.AddRoute("GET", tt.path, handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddRoute(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// 测试并发路由注册和匹配
func TestRouter_ConcurrentAccess(t *testing.T) {
	router := NewRouter()

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup

	// 并发注册路由（每个 goroutine 注册不同的路由）
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				path := fmt.Sprintf("/users/%d/%d/:id", id, j)
				router.AddRoute("GET", path, handler)
			}
		}(i)
	}
	wg.Wait()

	// 并发匹配路由
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				path := fmt.Sprintf("/users/%d/%d/123", id, j)
				result, ok := router.Match("GET", path)
				if ok {
					result.Release()
				}
			}
		}(i)
	}
	wg.Wait()
}

// 基准测试：静态路由匹配
func BenchmarkRouter_MatchStatic(b *testing.B) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	router.AddRoute("GET", "/api/v1/users", handler)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, _ := router.Match("GET", "/api/v1/users")
		result.Release()
	}
}

// 基准测试：参数路由匹配
func BenchmarkRouter_MatchParam(b *testing.B) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	router.AddRoute("GET", "/users/:id", handler)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, _ := router.Match("GET", "/users/123")
		result.Release()
	}
}

// 基准测试：参数路由匹配（禁用缓存）
func BenchmarkRouter_MatchParamNoCache(b *testing.B) {
	router := NewRouter()
	router.DisableHotCache()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	router.AddRoute("GET", "/users/:id", handler)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, _ := router.Match("GET", "/users/123")
		result.Release()
	}
}

// 基准测试：通配符路由匹配
func BenchmarkRouter_MatchWildcard(b *testing.B) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	router.AddRoute("GET", "/files/*path", handler)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, _ := router.Match("GET", "/files/docs/readme.md")
		result.Release()
	}
}

// 基准测试：缓存命中
func BenchmarkRouter_MatchCacheHit(b *testing.B) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	router.AddRoute("GET", "/users/:id", handler)

	// 预热缓存
	for i := 0; i < hotCacheThreshold+10; i++ {
		result, _ := router.Match("GET", "/users/123")
		result.Release()
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, _ := router.Match("GET", "/users/123")
		result.Release()
	}
}

// 测试CacheHitRate零除情况
func TestRouter_CacheHitRateZero(t *testing.T) {
	router := NewRouter()
	
	// 未进行任何匹配时
	hitRate := router.CacheHitRate()
	if hitRate != 0 {
		t.Errorf("hit rate = %f, want 0", hitRate)
	}
}

// 测试路径规范化场景
func TestRouter_PathNormalization(t *testing.T) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	
	router.AddRoute("GET", "/users", handler)
	
	// 测试需要规范化的路径
	tests := []string{
		"/users/",    // 结尾斜杠
		"/users//",   // 双斜杠
	}
	
	for _, path := range tests {
		result, ok := router.Match("GET", path)
		if !ok {
			t.Errorf("Match(%q) failed", path)
		} else {
			result.Release()
		}
	}
}
