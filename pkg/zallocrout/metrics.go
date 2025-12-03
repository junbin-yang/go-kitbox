package zallocrout

import (
	"sync/atomic"
	"unsafe"
)

// 路由性能指标
type RouterMetrics struct {
	CacheHits      uint64 // 缓存命中次数
	CacheMisses    uint64 // 缓存未命中次数
	StaticRoutes   uint64 // 静态路由数量
	ParamRoutes    uint64 // 参数路由数量
	WildcardRoutes uint64 // 通配符路由数量
	TotalMatches   uint64 // 总匹配次数
}

// 本地计数器（减少原子操作竞争）
type localCounter struct {
	cacheHits    uint64
	cacheMisses  uint64
	totalMatches uint64
	_padding     [40]byte // 缓存行填充，避免伪共享
}

// 异步 Metrics 收集器
type asyncMetricsCollector struct {
	global   *RouterMetrics
	local    [16]localCounter // 16 个本地计数器，减少竞争
	enabled  uint32           // 是否启用（原子操作）
}

// 创建异步 metrics 收集器
func newAsyncMetricsCollector(global *RouterMetrics) *asyncMetricsCollector {
	return &asyncMetricsCollector{
		global:  global,
		enabled: 1, // 默认启用
	}
}

// 快速路径：增加缓存命中次数（无锁本地计数）
//go:inline
func (c *asyncMetricsCollector) incrementCacheHits() {
	// 使用 goroutine ID 的低 4 位作为索引（通过栈地址近似）
	var dummy [1]byte
	idx := (uintptr(unsafe.Pointer(&dummy)) >> 4) & 15
	c.local[idx].cacheHits++
}

// 快速路径：增加缓存未命中次数（无锁本地计数）
//go:inline
func (c *asyncMetricsCollector) incrementCacheMisses() {
	var dummy [1]byte
	idx := (uintptr(unsafe.Pointer(&dummy)) >> 4) & 15
	c.local[idx].cacheMisses++
}

// 快速路径：增加总匹配次数（无锁本地计数）
//go:inline
func (c *asyncMetricsCollector) incrementTotalMatches() {
	var dummy [1]byte
	idx := (uintptr(unsafe.Pointer(&dummy)) >> 4) & 15
	c.local[idx].totalMatches++
}

// 定期聚合本地计数器到全局（由后台 goroutine 调用）
func (c *asyncMetricsCollector) flush() {
	for i := 0; i < 16; i++ {
		// 读取并重置本地计数器
		hits := atomic.SwapUint64(&c.local[i].cacheHits, 0)
		misses := atomic.SwapUint64(&c.local[i].cacheMisses, 0)
		matches := atomic.SwapUint64(&c.local[i].totalMatches, 0)

		// 批量更新全局计数器
		if hits > 0 {
			atomic.AddUint64(&c.global.CacheHits, hits)
		}
		if misses > 0 {
			atomic.AddUint64(&c.global.CacheMisses, misses)
		}
		if matches > 0 {
			atomic.AddUint64(&c.global.TotalMatches, matches)
		}
	}
}

// 启用 metrics 收集
func (c *asyncMetricsCollector) enable() {
	atomic.StoreUint32(&c.enabled, 1)
}

// 禁用 metrics 收集
func (c *asyncMetricsCollector) disable() {
	atomic.StoreUint32(&c.enabled, 0)
}

// 增加静态路由数量（路由注册时调用，不在热路径）
func (m *RouterMetrics) IncrementStaticRoutes() {
	atomic.AddUint64(&m.StaticRoutes, 1)
}

// 增加参数路由数量（路由注册时调用，不在热路径）
func (m *RouterMetrics) IncrementParamRoutes() {
	atomic.AddUint64(&m.ParamRoutes, 1)
}

// 增加通配符路由数量（路由注册时调用，不在热路径）
func (m *RouterMetrics) IncrementWildcardRoutes() {
	atomic.AddUint64(&m.WildcardRoutes, 1)
}

// 获取缓存命中率
func (m *RouterMetrics) CacheHitRate() float64 {
	hits := atomic.LoadUint64(&m.CacheHits)
	misses := atomic.LoadUint64(&m.CacheMisses)
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

// 获取指标快照
func (m *RouterMetrics) Snapshot() RouterMetrics {
	return RouterMetrics{
		CacheHits:      atomic.LoadUint64(&m.CacheHits),
		CacheMisses:    atomic.LoadUint64(&m.CacheMisses),
		StaticRoutes:   atomic.LoadUint64(&m.StaticRoutes),
		ParamRoutes:    atomic.LoadUint64(&m.ParamRoutes),
		WildcardRoutes: atomic.LoadUint64(&m.WildcardRoutes),
		TotalMatches:   atomic.LoadUint64(&m.TotalMatches),
	}
}

// 重置指标
func (m *RouterMetrics) Reset() {
	atomic.StoreUint64(&m.CacheHits, 0)
	atomic.StoreUint64(&m.CacheMisses, 0)
	atomic.StoreUint64(&m.StaticRoutes, 0)
	atomic.StoreUint64(&m.ParamRoutes, 0)
	atomic.StoreUint64(&m.WildcardRoutes, 0)
	atomic.StoreUint64(&m.TotalMatches, 0)
}
