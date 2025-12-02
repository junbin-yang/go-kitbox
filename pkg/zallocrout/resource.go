package zallocrout

import (
	"sync"

	"github.com/junbin-yang/go-kitbox/pkg/bytesconv"
)

// 资源池管理器
// 负责管理所有可复用资源的生命周期
type resourceManager struct {
	nodePool  sync.Pool // 路由节点池
	paramPool sync.Pool // 参数 Map 池
	segsPool  sync.Pool // 路径片段切片池（用于超长路径）
}

// 全局资源管理器实例
var globalResourceManager = &resourceManager{
	nodePool: sync.Pool{
		New: func() interface{} {
			return &RouteNode{
				staticChildren: make(map[string]*RouteNode, 4),
			}
		},
	},
	paramPool: sync.Pool{
		New: func() interface{} {
			return make(map[string]string, 4)
		},
	},
	segsPool: sync.Pool{
		New: func() interface{} {
			segs := make([]string, 0, 16)
			return &segs
		},
	},
}

// 获取路由节点（从池中）
func (rm *resourceManager) acquireNode() *RouteNode {
	n := rm.nodePool.Get().(*RouteNode)
	// 确保节点已初始化
	if n.staticChildren == nil {
		n.staticChildren = make(map[string]*RouteNode, 4)
	}
	return n
}

// 释放路由节点（归还到池中）
func (rm *resourceManager) releaseNode(n *RouteNode) {
	if n == nil {
		return
	}

	// 重置节点状态
	n.typ = StaticCompressed
	n.seg = n.seg[:0]
	n.handler = nil
	n.middlewares = n.middlewares[:0]
	n.paramChild = nil
	n.wildcardChild = nil
	n.paramName = n.paramName[:0]
	n.isWildcard = false
	n.hitCount = 0

	// 清理静态子节点（保留底层数组）
	for k := range n.staticChildren {
		delete(n.staticChildren, k)
	}

	rm.nodePool.Put(n)
}

// 获取参数 Map（从池中）
func (rm *resourceManager) acquireParamMap() map[string]string {
	return rm.paramPool.Get().(map[string]string)
}

// 释放参数 Map（归还到池中）
func (rm *resourceManager) releaseParamMap(params map[string]string) {
	if params == nil {
		return
	}

	// 清理 Map（保留底层数组）
	for k := range params {
		delete(params, k)
	}

	rm.paramPool.Put(params)
}

// 获取路径片段切片（从池中，用于超长路径）
func (rm *resourceManager) acquireSegsSlice() []string {
	segsPtr := rm.segsPool.Get().(*[]string)
	segs := *segsPtr
	return segs[:0]
}

// 释放路径片段切片（归还到池中）
func (rm *resourceManager) releaseSegsSlice(segs []string) {
	if segs == nil {
		return
	}

	// 清空切片（保留底层数组）
	segs = segs[:0]
	rm.segsPool.Put(&segs)
}

// 零拷贝转换：[]byte → string
// 复用项目中的 bytesconv 包
//
//go:inline
func unsafeString(b []byte) string {
	return bytesconv.BytesToString(b)
}

// 零拷贝转换：string → []byte
// 复用项目中的 bytesconv 包
//
//go:inline
func unsafeBytes(s string) []byte {
	return bytesconv.StringToBytes(s)
}

// 拷贝参数 Map（用于缓存）
func copyParamMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}

	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
