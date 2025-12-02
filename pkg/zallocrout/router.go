package zallocrout

import (
	"sync"
	"sync/atomic"
)

// 热点缓存阈值（命中次数超过此值才缓存）
const hotCacheThreshold = 100

// 空的释放函数（避免闭包分配）
var noopRelease = func() {}

// 路由器核心结构
type Router struct {
	roots          map[string]*RouteNode // HTTP 方法分层（GET/POST 等）
	rootsMu        sync.RWMutex          // 根节点读写锁
	resourceMgr    *resourceManager      // 资源管理器
	hotCache       *shardedMap           // 分片热点缓存
	metrics        *RouterMetrics        // 性能指标
	enableHotCache uint32                // 是否启用热点缓存（原子操作）
}

// 创建新的路由器
func NewRouter() *Router {
	return &Router{
		roots:          make(map[string]*RouteNode, 8),
		resourceMgr:    globalResourceManager,
		hotCache:       newShardedMap(),
		metrics:        &RouterMetrics{},
		enableHotCache: 1, // 默认启用
	}
}

// 添加路由
// method: HTTP 方法（GET/POST/PUT/DELETE 等）
// path: 路由路径（如 /api/v1/users/:id）
// handler: 处理函数
// middlewares: 中间件列表
//
//go:inline
func (r *Router) AddRoute(method, path string, handler HandlerFunc, middlewares ...Middleware) error {
	// 1. 路由预检查
	if err := validateRoute(path); err != nil {
		return err
	}

	// 2. 获取/创建方法根节点
	r.rootsMu.Lock()
	root, exists := r.roots[method]
	if !exists {
		root = r.resourceMgr.acquireNode()
		r.roots[method] = root
	}
	r.rootsMu.Unlock()

	// 3. 路径拆分（栈分配优先）
	var segs [MaxParams]string
	segsSlice := splitPathToCompressedSegs(path, segs[:0])

	// 4. 逐层插入 Trie 树
	current := root
	for _, seg := range segsSlice {
		// 静态路由：需要加锁避免并发写入冲突
		if isStaticSeg(seg) {
			// 加锁检查和插入（避免并发 map 读写冲突）
			current.mu.Lock()
			if child, ok := current.staticChildren[seg]; ok {
				current.mu.Unlock()
				current = child
				continue
			}

			// 创建新的静态节点
			child := r.resourceMgr.acquireNode()
			child.typ = StaticCompressed
			child.seg = []byte(seg)
			current.staticChildren[seg] = child
			current.mu.Unlock()
			current = child
			continue
		}

		// 参数路由
		if isParamSeg(seg) {
			// 加锁检查和插入
			current.mu.Lock()
			if current.paramChild != nil {
				child := current.paramChild
				current.mu.Unlock()
				current = child
				continue
			}

			// 创建新的参数节点
			child := r.resourceMgr.acquireNode()
			child.typ = ParamNode
			child.seg = []byte(seg)
			child.paramName = []byte(seg[1:]) // 去掉 ':'
			current.paramChild = child
			current.mu.Unlock()
			current = child
			continue
		}

		// 通配符路由
		if isWildcardSeg(seg) {
			// 加锁检查和插入
			current.mu.Lock()
			if current.wildcardChild != nil {
				child := current.wildcardChild
				current.mu.Unlock()
				current = child
				break
			}

			// 创建新的通配符节点
			child := r.resourceMgr.acquireNode()
			child.typ = WildcardNode
			child.seg = []byte(seg)
			child.isWildcard = true
			current.wildcardChild = child
			current.mu.Unlock()
			current = child
			break
		}
	}

	// 5. 绑定处理器和中间件
	current.setHandler(handler, middlewares)

	// 6. 更新指标
	switch current.typ {
	case StaticCompressed:
		r.metrics.IncrementStaticRoutes()
	case ParamNode:
		r.metrics.IncrementParamRoutes()
	case WildcardNode:
		r.metrics.IncrementWildcardRoutes()
	}

	return nil
}

// 匹配路由
// method: HTTP 方法
// path: 请求路径
//
//go:inline
func (r *Router) Match(method, path string) (MatchResult, bool) {
	r.metrics.IncrementTotalMatches()

	// 1. 热点缓存检查（如果启用）
	if atomic.LoadUint32(&r.enableHotCache) == 1 {
		cacheKey := method + ":" + path
		if cacheVal, ok := r.hotCache.Load(cacheKey); ok {
			r.metrics.IncrementCacheHits()

			// 构建参数列表
			var result MatchResult
			for k, v := range cacheVal.paramTemplate {
				if result.paramCount < MaxParams {
					result.paramPairs[result.paramCount] = paramPair{key: k, value: v}
					result.paramCount++
				}
			}
			result.Handler = cacheVal.handler
			result.Middlewares = cacheVal.middlewares
			result.Release = noopRelease
			return result, true
		}
		r.metrics.IncrementCacheMisses()
	}

	// 2. 路径规范化（零分配优化：大多数路径不需要规范化）
	normalizedPath := path
	if needsNormalization(path) {
		pathBytes := []byte(path)
		pathBytes = normalizePathBytes(pathBytes)
		if len(pathBytes) == 0 {
			return MatchResult{}, false
		}
		normalizedPath = unsafeString(pathBytes)
	}

	// 3. 路径拆分（栈分配优先）
	var segs [MaxParams]string
	segsSlice := splitPathToCompressedSegs(normalizedPath, segs[:0])

	// 4. 快速定位方法根节点
	r.rootsMu.RLock()
	root, exists := r.roots[method]
	r.rootsMu.RUnlock()

	if !exists {
		return MatchResult{}, false
	}

	// 5. 核心匹配流程（零分配参数存储）
	current := root
	var result MatchResult
	pathPos := 1 // 跳过开头的 '/'

	for _, seg := range segsSlice {
		// 静态路由：无锁 O(1) 匹配
		if child, ok := current.findStaticChild(seg); ok {
			current = child
			pathPos += len(seg) + 1 // +1 for '/'
			continue
		}

		// 参数节点匹配
		if paramChild := current.findParamChild(); paramChild != nil {
			if result.paramCount < MaxParams {
				result.paramPairs[result.paramCount] = paramPair{
					key:   unsafeString(paramChild.paramName),
					value: seg,
				}
				result.paramCount++
			}
			current = paramChild
			pathPos += len(seg) + 1
			continue
		}

		// 通配符节点匹配
		if wildcardChild := current.findWildcardChild(); wildcardChild != nil {
			// 直接使用原始路径的剩余部分（零分配）
			remaining := normalizedPath[pathPos:]
			if result.paramCount < MaxParams {
				result.paramPairs[result.paramCount] = paramPair{key: "*", value: remaining}
				result.paramCount++
			}
			current = wildcardChild
			break
		}

		// 未找到匹配
		return MatchResult{}, false
	}

	// 6. 检查处理器是否存在
	handler, middlewares := current.getHandler()
	if handler == nil {
		return MatchResult{}, false
	}

	// 7. 热点缓存更新（原子操作 + 分片 Map）
	if atomic.LoadUint32(&r.enableHotCache) == 1 && !current.isWildcard {
		hitCount := current.incrementHitCount()
		if hitCount > hotCacheThreshold {
			cacheKey := method + ":" + path
			cacheData := &cacheEntry{
				handler:     handler,
				middlewares: middlewares,
			}
			// 缓存参数模板（如果有参数）
			if result.paramCount > 0 {
				cacheData.paramTemplate = make(map[string]string, result.paramCount)
				for i := 0; i < result.paramCount; i++ {
					cacheData.paramTemplate[result.paramPairs[i].key] = result.paramPairs[i].value
				}
			}
			r.hotCache.Store(cacheKey, cacheData)
		}
	}

	// 8. 返回结果（零分配）
	result.Handler = handler
	result.Middlewares = middlewares
	result.Release = noopRelease
	return result, true
}

// 获取性能指标
func (r *Router) Metrics() RouterMetrics {
	return r.metrics.Snapshot()
}

// 获取缓存命中率
func (r *Router) CacheHitRate() float64 {
	return r.metrics.CacheHitRate()
}

// 启用热点缓存
func (r *Router) EnableHotCache() {
	atomic.StoreUint32(&r.enableHotCache, 1)
}

// 禁用热点缓存
func (r *Router) DisableHotCache() {
	atomic.StoreUint32(&r.enableHotCache, 0)
}

// 清空热点缓存
func (r *Router) ClearHotCache() {
	r.hotCache.Clear()
}

// 获取缓存统计信息
func (r *Router) CacheStats() (totalEntries int64, shardDistribution [shardCount]int64) {
	return r.hotCache.Stats()
}
