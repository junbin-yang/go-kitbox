package zallocrout

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// 分片数量（16 分片，降低竞争）
const shardCount = 16

// 每个分片的最大缓存条目数（LRU 淘汰）
const maxEntriesPerShard = 1000

// 缓存条目
type cacheEntry struct {
	handler       HandlerFunc       // 处理函数
	middlewares   []Middleware      // 预编译中间件链
	paramTemplate map[string]string // 参数模板
	timestamp     int64             // 时间戳（LRU 淘汰用）
	hitCount      uint64            // 命中次数
}

// 分片热点缓存
// 使用 16 个 sync.Map 分片，解决 sync.Map 写入瓶颈
type shardedMap struct {
	shards     [shardCount]sync.Map // 16 个分片
	entryCount [shardCount]int64    // 每个分片的条目数
}

// 创建分片缓存
func newShardedMap() *shardedMap {
	return &shardedMap{}
}

// 计算分片索引（使用 FNV-1a 哈希）
func (sm *shardedMap) getShard(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % shardCount)
}

// 加载缓存条目
func (sm *shardedMap) Load(key string) (*cacheEntry, bool) {
	shardIdx := sm.getShard(key)
	val, ok := sm.shards[shardIdx].Load(key)
	if !ok {
		return nil, false
	}

	entry := val.(*cacheEntry)
	// 更新命中次数和时间戳
	atomic.AddUint64(&entry.hitCount, 1)
	atomic.StoreInt64(&entry.timestamp, time.Now().UnixNano())

	return entry, true
}

// 存储缓存条目
func (sm *shardedMap) Store(key string, entry *cacheEntry) {
	shardIdx := sm.getShard(key)

	// 检查分片是否已满，需要淘汰
	count := atomic.LoadInt64(&sm.entryCount[shardIdx])
	if count >= maxEntriesPerShard {
		sm.evictLRU(shardIdx)
	}

	// 存储条目
	entry.timestamp = time.Now().UnixNano()
	sm.shards[shardIdx].Store(key, entry)
	atomic.AddInt64(&sm.entryCount[shardIdx], 1)
}

// LRU 淘汰（淘汰最久未使用的 10% 条目）
func (sm *shardedMap) evictLRU(shardIdx int) {
	type entryWithKey struct {
		key       string
		timestamp int64
	}

	// 收集所有条目
	entries := make([]entryWithKey, 0, maxEntriesPerShard)
	sm.shards[shardIdx].Range(func(key, value interface{}) bool {
		entry := value.(*cacheEntry)
		entries = append(entries, entryWithKey{
			key:       key.(string),
			timestamp: atomic.LoadInt64(&entry.timestamp),
		})
		return true
	})

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

	// 删除最旧的条目
	for i := 0; i < evictCount; i++ {
		sm.shards[shardIdx].Delete(entries[i].key)
		atomic.AddInt64(&sm.entryCount[shardIdx], -1)
	}
}

// 删除缓存条目
func (sm *shardedMap) Delete(key string) {
	shardIdx := sm.getShard(key)
	sm.shards[shardIdx].Delete(key)
	atomic.AddInt64(&sm.entryCount[shardIdx], -1)
}

// 清空所有缓存
func (sm *shardedMap) Clear() {
	for i := 0; i < shardCount; i++ {
		sm.shards[i].Range(func(key, value interface{}) bool {
			sm.shards[i].Delete(key)
			return true
		})
		atomic.StoreInt64(&sm.entryCount[i], 0)
	}
}

// 获取缓存统计信息
func (sm *shardedMap) Stats() (totalEntries int64, shardDistribution [shardCount]int64) {
	for i := 0; i < shardCount; i++ {
		count := atomic.LoadInt64(&sm.entryCount[i])
		shardDistribution[i] = count
		totalEntries += count
	}
	return
}
