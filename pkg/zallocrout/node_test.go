package zallocrout

import (
	"net/http"
	"sync"
	"testing"
)

// 测试节点插入和查找（静态）
func TestRouteNode_StaticChild(t *testing.T) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}

	// 插入静态子节点
	child := &RouteNode{
		typ: StaticCompressed,
		seg: []byte("users"),
	}
	node.insertChild("users", child)

	// 查找静态子节点
	found, ok := node.findStaticChild("users")
	if !ok {
		t.Fatal("static child not found")
	}
	if found != child {
		t.Error("found wrong child")
	}

	// 查找不存在的子节点
	_, ok = node.findStaticChild("posts")
	if ok {
		t.Error("should not find non-existent child")
	}
}

// 测试节点插入和查找（参数）
func TestRouteNode_ParamChild(t *testing.T) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}

	// 插入参数子节点
	child := &RouteNode{
		typ:       ParamNode,
		seg:       []byte(":id"),
		paramName: []byte("id"),
	}
	node.insertChild(":id", child)

	// 查找参数子节点
	found := node.findParamChild()
	if found == nil {
		t.Fatal("param child not found")
	}
	if found != child {
		t.Error("found wrong child")
	}
}

// 测试节点插入和查找（通配符）
func TestRouteNode_WildcardChild(t *testing.T) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}

	// 插入通配符子节点
	child := &RouteNode{
		typ:        WildcardNode,
		seg:        []byte("*path"),
		isWildcard: true,
	}
	node.insertChild("*path", child)

	// 查找通配符子节点
	found := node.findWildcardChild()
	if found == nil {
		t.Fatal("wildcard child not found")
	}
	if found != child {
		t.Error("found wrong child")
	}
}

// 测试设置和获取处理器
func TestRouteNode_Handler(t *testing.T) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}

	// 定义处理器和中间件
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		// 测试处理器
	}
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

	// 设置处理器
	node.setHandler(handler, []Middleware{middleware1, middleware2})

	// 获取处理器
	gotHandler, gotMiddlewares := node.getHandler()
	if gotHandler == nil {
		t.Fatal("handler not set")
	}
	if len(gotMiddlewares) != 2 {
		t.Errorf("middlewares length = %d, want 2", len(gotMiddlewares))
	}
}

// 测试命中计数
func TestRouteNode_HitCount(t *testing.T) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}

	// 初始计数应该为 0
	if count := node.getHitCount(); count != 0 {
		t.Errorf("initial hit count = %d, want 0", count)
	}

	// 增加计数
	count1 := node.incrementHitCount()
	if count1 != 1 {
		t.Errorf("first increment = %d, want 1", count1)
	}

	count2 := node.incrementHitCount()
	if count2 != 2 {
		t.Errorf("second increment = %d, want 2", count2)
	}

	// 获取计数
	if count := node.getHitCount(); count != 2 {
		t.Errorf("final hit count = %d, want 2", count)
	}
}

// 测试并发访问节点
func TestRouteNode_ConcurrentAccess(t *testing.T) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// 并发写入
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				node.setHandler(handler, nil)
			}
		}()
	}

	// 并发读取
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				node.getHandler()
				node.incrementHitCount()
			}
		}()
	}

	wg.Wait()

	// 验证最终计数
	finalCount := node.getHitCount()
	expectedCount := uint32(goroutines * iterations)
	if finalCount != expectedCount {
		t.Errorf("final hit count = %d, want %d", finalCount, expectedCount)
	}
}

// 测试节点类型
func TestNodeType(t *testing.T) {
	tests := []struct {
		name string
		typ  NodeType
	}{
		{"StaticCompressed", StaticCompressed},
		{"ParamNode", ParamNode},
		{"WildcardNode", WildcardNode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &RouteNode{
				typ:            tt.typ,
				staticChildren: make(map[string]*RouteNode),
			}
			if node.typ != tt.typ {
				t.Errorf("node type = %v, want %v", node.typ, tt.typ)
			}
		})
	}
}

// 测试匹配结果
func TestMatchResult(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {}
	released := false

	result := MatchResult{
		Handler:    handler,
		paramPairs: [MaxParams]paramPair{{key: "id", value: "123"}},
		paramCount: 1,
		Release: func() {
			released = true
		},
	}

	if result.Handler == nil {
		t.Error("handler is nil")
	}
	id, ok := result.GetParam("id")
	if !ok || id != "123" {
		t.Error("params not set correctly")
	}

	// 调用释放函数
	result.Release()
	if !released {
		t.Error("release function not called")
	}
}

// 基准测试：静态子节点查找
func BenchmarkRouteNode_FindStaticChild(b *testing.B) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}
	child := &RouteNode{typ: StaticCompressed}
	node.staticChildren["users"] = child

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = node.findStaticChild("users")
	}
}

// 基准测试：参数子节点查找
func BenchmarkRouteNode_FindParamChild(b *testing.B) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
		paramChild:     &RouteNode{typ: ParamNode},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = node.findParamChild()
	}
}

// 基准测试：命中计数增加
func BenchmarkRouteNode_IncrementHitCount(b *testing.B) {
	node := &RouteNode{
		staticChildren: make(map[string]*RouteNode),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		node.incrementHitCount()
	}
}

// 测试GetParam未找到的情况
func TestMatchResult_GetParamNotFound(t *testing.T) {
	result := MatchResult{
		paramPairs: [MaxParams]paramPair{{key: "id", value: "123"}},
		paramCount: 1,
	}

	// 查找不存在的参数
	_, ok := result.GetParam("notfound")
	if ok {
		t.Error("should not find non-existent param")
	}

	// 查找存在的参数
	val, ok := result.GetParam("id")
	if !ok || val != "123" {
		t.Error("should find existing param")
	}
}
