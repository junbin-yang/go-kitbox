package zallocrout

import "sync/atomic"

// 路由性能指标
type RouterMetrics struct {
	CacheHits      uint64 // 缓存命中次数
	CacheMisses    uint64 // 缓存未命中次数
	StaticRoutes   uint64 // 静态路由数量
	ParamRoutes    uint64 // 参数路由数量
	WildcardRoutes uint64 // 通配符路由数量
	TotalMatches   uint64 // 总匹配次数
}

// 增加缓存命中次数
func (m *RouterMetrics) IncrementCacheHits() {
	atomic.AddUint64(&m.CacheHits, 1)
}

// 增加缓存未命中次数
func (m *RouterMetrics) IncrementCacheMisses() {
	atomic.AddUint64(&m.CacheMisses, 1)
}

// 增加静态路由数量
func (m *RouterMetrics) IncrementStaticRoutes() {
	atomic.AddUint64(&m.StaticRoutes, 1)
}

// 增加参数路由数量
func (m *RouterMetrics) IncrementParamRoutes() {
	atomic.AddUint64(&m.ParamRoutes, 1)
}

// 增加通配符路由数量
func (m *RouterMetrics) IncrementWildcardRoutes() {
	atomic.AddUint64(&m.WildcardRoutes, 1)
}

// 增加总匹配次数
func (m *RouterMetrics) IncrementTotalMatches() {
	atomic.AddUint64(&m.TotalMatches, 1)
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
