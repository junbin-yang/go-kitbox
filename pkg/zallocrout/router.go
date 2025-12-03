package zallocrout

import (
	"context"
	"sync"
	"sync/atomic"
)

// 热点缓存阈值（命中次数超过此值才缓存）
const hotCacheThreshold = 100

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

	// 4. 收集参数名称（使用字节切片避免转换开销）
	var paramNames [][]byte
	for _, seg := range segsSlice {
		if isParamSeg(seg) {
			// 去掉 ':' 前缀，直接存储字节切片
			paramNames = append(paramNames, []byte(seg[1:]))
		}
	}

	// 5. 逐层插入 Trie 树
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

	// 6. 绑定处理器和中间件，并存储参数名称列表
	current.setHandlerWithParams(handler, middlewares, paramNames)

	// 7. 更新指标
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
// parent: 父 context
//
//go:inline
func (r *Router) Match(method, path string, parent context.Context) (context.Context, HandlerFunc, []Middleware, bool) {
	r.metrics.IncrementTotalMatches()

	// 1. 热点缓存检查（如果启用）
	if atomic.LoadUint32(&r.enableHotCache) == 1 {
		cacheKey := method + path
		if cacheVal, ok := r.hotCache.Load(cacheKey); ok {
			r.metrics.IncrementCacheHits()

			// 构建参数数组（优化：预先检查是否有参数）
			var paramPairs [MaxParams]paramPair
			paramCount := len(cacheVal.paramTemplate)
			if paramCount > 0 {
				i := 0
				for k, v := range cacheVal.paramTemplate {
					if i < MaxParams {
						paramPairs[i] = paramPair{key: k, value: v}
						i++
					}
				}
			}

			// 从池中获取 context（使用指针避免数组拷贝）
			ctx := acquireContext(parent, &paramPairs, paramCount)
			return ctx, cacheVal.handler, cacheVal.middlewares, true
		}
		r.metrics.IncrementCacheMisses()
	}

	// 2. 路径规范化（零分配优化：大多数路径不需要规范化）
	normalizedPath := path
	if needsNormalization(path) {
		pathBytes := []byte(path)
		pathBytes = normalizePathBytes(pathBytes)
		if len(pathBytes) == 0 {
			return nil, nil, nil, false
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
		return nil, nil, nil, false
	}

	// 5. 核心匹配流程（零分配参数存储）
	current := root
	var paramPairs [MaxParams]paramPair
	paramCount := 0
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
			if paramCount < MaxParams {
				paramPairs[paramCount] = paramPair{
					key:   unsafeString(paramChild.paramName),
					value: seg,
				}
				paramCount++
			}
			current = paramChild
			pathPos += len(seg) + 1
			continue
		}

		// 通配符节点匹配
		if wildcardChild := current.findWildcardChild(); wildcardChild != nil {
			// 直接使用原始路径的剩余部分（零分配）
			remaining := normalizedPath[pathPos:]
			if paramCount < MaxParams {
				paramPairs[paramCount] = paramPair{key: "*", value: remaining}
				paramCount++
			}
			current = wildcardChild
			break
		}

		// 未找到匹配
		return nil, nil, nil, false
	}

	// 6. 检查处理器是否存在
	handler, middlewares, routeParamNames := current.getHandler()
	if handler == nil {
		return nil, nil, nil, false
	}

	// 7. 使用路由定义的参数名称重新映射参数
	// 如果路由有自己的参数名称列表，使用它；否则使用遍历时收集的名称
	if len(routeParamNames) > 0 && len(routeParamNames) == paramCount {
		// 重新映射参数名称（使用 unsafeString 避免内存分配）
		for i := 0; i < paramCount && i < len(routeParamNames); i++ {
			paramPairs[i].key = unsafeString(routeParamNames[i])
		}
	}

	// 8. 热点缓存更新（原子操作 + 分片 Map）
	if atomic.LoadUint32(&r.enableHotCache) == 1 && !current.isWildcard {
		hitCount := current.incrementHitCount()
		if hitCount > hotCacheThreshold {
			cacheKey := method + path
			cacheData := &cacheEntry{
				handler:     handler,
				middlewares: middlewares,
			}
			// 缓存参数模板（如果有参数）
			if paramCount > 0 {
				cacheData.paramTemplate = make(map[string]string, paramCount)
				for i := 0; i < paramCount; i++ {
					cacheData.paramTemplate[paramPairs[i].key] = paramPairs[i].value
				}
			}
			r.hotCache.Store(cacheKey, cacheData)
		}
	}

	// 9. 从池中获取 context 并返回（使用指针避免数组拷贝）
	ctx := acquireContext(parent, &paramPairs, paramCount)
	return ctx, handler, middlewares, true
}

// 获取性能指标
func (r *Router) Metrics() RouterMetrics {
	return r.metrics.Snapshot()
}

// 重置性能指标
func (r *Router) ResetMetrics() {
	r.metrics.Reset()
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
