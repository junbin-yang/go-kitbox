package zallocrout

import (
	"sync"
	"testing"
)

// 测试节点池获取和释放
func TestResourceManager_NodePool(t *testing.T) {
	rm := globalResourceManager

	// 获取节点
	n1 := rm.acquireNode()
	if n1 == nil {
		t.Fatal("acquireNode returned nil")
	}
	if n1.staticChildren == nil {
		t.Fatal("node staticChildren not initialized")
	}

	// 修改节点状态
	n1.typ = ParamNode
	n1.seg = []byte("test")
	n1.hitCount = 100

	// 释放节点
	rm.releaseNode(n1)

	// 再次获取节点（应该是同一个节点，但已重置）
	n2 := rm.acquireNode()
	if n2.typ != StaticCompressed {
		t.Errorf("node type not reset, got %v", n2.typ)
	}
	if len(n2.seg) != 0 {
		t.Errorf("node seg not reset, got %v", n2.seg)
	}
	if n2.hitCount != 0 {
		t.Errorf("node hitCount not reset, got %v", n2.hitCount)
	}

	rm.releaseNode(n2)
}

// 测试路径片段切片池获取和释放
func TestResourceManager_SegsSlicePool(t *testing.T) {
	rm := globalResourceManager

	// 获取切片
	segs1 := rm.acquireSegsSlice()
	if segs1 == nil {
		t.Fatal("acquireSegsSlice returned nil")
	}

	// 添加片段
	segs1 = append(segs1, "api", "v1", "users")

	// 释放切片
	rm.releaseSegsSlice(segs1)

	// 再次获取切片（应该已清空）
	segs2 := rm.acquireSegsSlice()
	if len(segs2) != 0 {
		t.Errorf("segs slice not reset, got %v", segs2)
	}

	rm.releaseSegsSlice(segs2)
}

// 测试并发获取和释放节点
func TestResourceManager_ConcurrentNodePool(t *testing.T) {
	rm := globalResourceManager
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				n := rm.acquireNode()
				n.typ = ParamNode
				n.seg = []byte("test")
				rm.releaseNode(n)
			}
		}()
	}

	wg.Wait()
}

// 测试零拷贝转换：[]byte → string
func TestUnsafeString(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte("hello"), "hello"},
		{[]byte(""), ""},
		{[]byte("api/v1/users"), "api/v1/users"},
	}

	for _, tt := range tests {
		result := unsafeString(tt.input)
		if result != tt.expected {
			t.Errorf("unsafeString(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// 测试零拷贝转换：string → []byte
func TestUnsafeBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{"hello", []byte("hello")},
		{"", nil},
		{"api/v1/users", []byte("api/v1/users")},
	}

	for _, tt := range tests {
		result := unsafeBytes(tt.input)
		if string(result) != string(tt.expected) {
			t.Errorf("unsafeBytes(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// 测试释放 nil 节点
func TestResourceManager_ReleaseNilNode(t *testing.T) {
	rm := globalResourceManager
	// 不应该 panic
	rm.releaseNode(nil)
}

// 测试释放 nil 切片
func TestResourceManager_ReleaseNilSegsSlice(t *testing.T) {
	rm := globalResourceManager
	// 不应该 panic
	rm.releaseSegsSlice(nil)
}

// 基准测试：节点池获取和释放
func BenchmarkResourceManager_NodePool(b *testing.B) {
	rm := globalResourceManager
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n := rm.acquireNode()
		rm.releaseNode(n)
	}
}

// 基准测试：零拷贝转换
func BenchmarkUnsafeString(b *testing.B) {
	data := []byte("api/v1/users/123/posts")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = unsafeString(data)
	}
}

// 测试acquireNode初始化
func TestResourceManager_AcquireNodeInit(t *testing.T) {
	mgr := &resourceManager{
		nodePool: sync.Pool{
			New: func() interface{} {
				return &RouteNode{}
			},
		},
	}

	node := mgr.acquireNode()
	if node == nil {
		t.Fatal("acquireNode returned nil")
	}
	if node.staticChildren == nil {
		t.Error("staticChildren not initialized")
	}
}
