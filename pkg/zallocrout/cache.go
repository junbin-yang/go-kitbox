package zallocrout

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgryski/go-wyhash"
)

// 分片数量（16 分片，降低竞争）
const shardCount = 16

// 每个分片的最大缓存条目数（LRU 淘汰）
const maxEntriesPerShard = 1000

// 缓存条目
type cacheEntry struct {
	handler     HandlerFunc          // 处理函数
	middlewares []Middleware         // 预编译中间件链
	paramPairs  [MaxParams]paramPair // 参数数组
	paramCount  int                  // 参数数量
	timestamp   int64                // 时间戳（LRU 淘汰用）
	hitCount    uint64               // 命中次数
}

// 复合缓存键
type cacheKey struct {
	method string
	path   string
}

// 分片缓存
type cacheShard struct {
	mu      sync.Mutex  // 写锁（只在写入时使用）
	entries atomic.Pointer[map[cacheKey]*cacheEntry]  // 原子指针，无锁读
	count   int64
}

// 分片热点缓存
type shardedMap struct {
	shards [shardCount]cacheShard
}

// 创建分片缓存
func newShardedMap() *shardedMap {
	sm := &shardedMap{}
	for i := 0; i < shardCount; i++ {
		m := make(map[cacheKey]*cacheEntry, 64)
		sm.shards[i].entries.Store(&m)
	}
	return sm
}

// 计算分片索引
func (sm *shardedMap) getShard(key cacheKey) int {
	h := wyhash.Hash([]byte(key.method), 0)
	h ^= wyhash.Hash([]byte(key.path), 0)
	return int(h % shardCount)
}

//go:inline
func (sm *shardedMap) LoadWithMethodPath(method, path string) (*cacheEntry, bool) {
	key := cacheKey{method: method, path: path}

	// 内联哈希计算
	h := wyhash.Hash([]byte(method), 0)
	h ^= wyhash.Hash([]byte(path), 0)
	shardIdx := int(h % shardCount)

	shard := &sm.shards[shardIdx]

	// 无锁读取：直接加载 map 指针
	entriesPtr := shard.entries.Load()
	if entriesPtr == nil {
		return nil, false
	}

	entry, ok := (*entriesPtr)[key]
	return entry, ok
}

// 基于 Copy-on-Write 写入
func (sm *shardedMap) StoreWithMethodPath(method, path string, entry *cacheEntry) {
	key := cacheKey{method: method, path: path}
	shardIdx := sm.getShard(key)

	shard := &sm.shards[shardIdx]

	// 检查分片是否已满，需要淘汰
	if atomic.LoadInt64(&shard.count) >= maxEntriesPerShard {
		sm.evictLRU(shardIdx)
	}

	// Copy-on-Write：创建新 map 副本
	shard.mu.Lock()
	oldEntriesPtr := shard.entries.Load()
	oldEntries := *oldEntriesPtr

	// 创建新 map（复制旧数据 + 新数据）
	newEntries := make(map[cacheKey]*cacheEntry, len(oldEntries)+1)
	for k, v := range oldEntries {
		newEntries[k] = v
	}

	// 添加新条目
	entry.timestamp = time.Now().UnixNano()
	newEntries[key] = entry

	// 原子更新指针
	shard.entries.Store(&newEntries)
	atomic.AddInt64(&shard.count, 1)
	shard.mu.Unlock()
}

// LRU 淘汰（淘汰最久未使用的 10% 条目）
func (sm *shardedMap) evictLRU(shardIdx int) {
	type entryWithKey struct {
		key       cacheKey
		timestamp int64
	}

	shard := &sm.shards[shardIdx]

	// 无锁读取当前 map
	entriesPtr := shard.entries.Load()
	if entriesPtr == nil {
		return
	}
	oldEntries := *entriesPtr

	// 收集所有条目
	entries := make([]entryWithKey, 0, len(oldEntries))
	for key, entry := range oldEntries {
		entries = append(entries, entryWithKey{
			key:       key,
			timestamp: atomic.LoadInt64(&entry.timestamp),
		})
	}

	// 如果条目数不足，不淘汰
	if len(entries) < maxEntriesPerShard {
		return
	}

	// 按时间戳排序（冒泡排序，只需找到最旧的 10%）
	evictCount := maxEntriesPerShard / 10
	if evictCount == 0 {
		evictCount = 1
	}

	// 找到最旧的 evictCount 个条目
	for i := 0; i < evictCount; i++ {
		minIdx := i
		for j := i + 1; j < len(entries); j++ {
			if entries[j].timestamp < entries[minIdx].timestamp {
				minIdx = j
			}
		}
		if minIdx != i {
			entries[i], entries[minIdx] = entries[minIdx], entries[i]
		}
	}

	// 删除最旧的条目（Copy-on-Write）
	shard.mu.Lock()
	newEntries := make(map[cacheKey]*cacheEntry, len(oldEntries)-evictCount)
	for k, v := range oldEntries {
		// 检查是否在淘汰列表中
		shouldEvict := false
		for i := 0; i < evictCount; i++ {
			if entries[i].key == k {
				shouldEvict = true
				break
			}
		}
		if !shouldEvict {
			newEntries[k] = v
		}
	}
	shard.entries.Store(&newEntries)
	atomic.AddInt64(&shard.count, -int64(evictCount))
	shard.mu.Unlock()
}

// 清空所有缓存
func (sm *shardedMap) Clear() {
	for i := 0; i < shardCount; i++ {
		shard := &sm.shards[i]
		shard.mu.Lock()
		emptyMap := make(map[cacheKey]*cacheEntry)
		shard.entries.Store(&emptyMap)
		atomic.StoreInt64(&shard.count, 0)
		shard.mu.Unlock()
	}
}

// 获取缓存统计信息
func (sm *shardedMap) Stats() (totalEntries int64, shardDistribution [shardCount]int64) {
	for i := 0; i < shardCount; i++ {
		count := atomic.LoadInt64(&sm.shards[i].count)
		shardDistribution[i] = count
		totalEntries += count
	}
	return
}
