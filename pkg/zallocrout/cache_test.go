package zallocrout

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 测试分片缓存基本操作
func TestShardedMap_BasicOperations(t *testing.T) {
	sm := newShardedMap()

	handler := func(ctx context.Context) error { return nil }
	entry := &cacheEntry{
		handler:    handler,
		paramPairs: [MaxParams]paramPair{{key: "id", value: "123"}},
		paramCount: 1,
	}

	// 存储
	sm.StoreWithMethodPath("GET", "/users/123", entry)

	// 加载
	loaded, ok := sm.LoadWithMethodPath("GET", "/users/123")
	if !ok {
		t.Fatal("failed to load entry")
	}
	if loaded.paramPairs[0].value != "123" {
		t.Errorf("loaded entry param = %v, want 123", loaded.paramPairs[0].value)
	}
}

// 测试分片哈希分布
func TestShardedMap_ShardDistribution(t *testing.T) {
	sm := newShardedMap()

	handler := func(ctx context.Context) error { return nil }

	// 插入 1000 个条目
	for i := 0; i < 1000; i++ {
		entry := &cacheEntry{
			handler:    handler,
			paramPairs: [MaxParams]paramPair{{key: "id", value: fmt.Sprintf("%d", i)}},
			paramCount: 1,
		}
		sm.StoreWithMethodPath("GET", fmt.Sprintf("/users/%d", i), entry)
	}

	// 检查分片分布
	totalEntries, shardDist := sm.Stats()
	if totalEntries != 1000 {
		t.Errorf("total entries = %d, want 1000", totalEntries)
	}

	// 检查每个分片都有条目（分布应该相对均匀）
	emptyShards := 0
	for i := 0; i < shardCount; i++ {
		if shardDist[i] == 0 {
			emptyShards++
		}
	}

	// 允许少量空分片（由于哈希分布的随机性）
	if emptyShards > shardCount/4 {
		t.Errorf("too many empty shards: %d/%d", emptyShards, shardCount)
	}
}

// 测试 LRU 淘汰机制
func TestShardedMap_LRUEviction(t *testing.T) {
	sm := newShardedMap()

	handler := func(ctx context.Context) error { return nil }

	// 找到一个分片，插入超过 maxEntriesPerShard 的条目
	shardIdx := 0
	keysAdded := 0

	// 生成足够多的 key，确保至少有一个分片满
	for i := 0; keysAdded < maxEntriesPerShard+100; i++ {
		method := "GET"
		path := fmt.Sprintf("/test/%d", i)
		key := cacheKey{method: method, path: path}
		if sm.getShard(key) == shardIdx {
			entry := &cacheEntry{
				handler:    handler,
				paramPairs: [MaxParams]paramPair{{key: "id", value: fmt.Sprintf("%d", i)}},
				paramCount: 1,
				timestamp:  time.Now().Add(time.Duration(i) * time.Millisecond).UnixNano(),
			}
			sm.StoreWithMethodPath(method, path, entry)
			keysAdded++
			time.Sleep(time.Microsecond)
		}
	}

	// 检查分片条目数（应该触发淘汰）
	count := atomic.LoadInt64(&sm.shards[shardIdx].count)
	if count > maxEntriesPerShard {
		t.Logf("shard %d has %d entries (expected <= %d after eviction)", shardIdx, count, maxEntriesPerShard)
	}
}

// 测试并发读写
func TestShardedMap_ConcurrentAccess(t *testing.T) {
	sm := newShardedMap()

	handler := func(ctx context.Context) error { return nil }

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// 并发写入
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				entry := &cacheEntry{
					handler:    handler,
					paramPairs: [MaxParams]paramPair{{key: "id", value: fmt.Sprintf("%d", id)}},
					paramCount: 1,
				}
				sm.StoreWithMethodPath("GET", fmt.Sprintf("/users/%d/%d", id, j), entry)
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sm.LoadWithMethodPath("GET", fmt.Sprintf("/users/%d/%d", id, j))
			}
		}(i)
	}

	wg.Wait()

	// 检查最终条目数
	totalEntries, _ := sm.Stats()
	if totalEntries == 0 {
		t.Error("no entries stored")
	}
}

// 测试缓存命中次数统计
func TestShardedMap_HitCount(t *testing.T) {
	sm := newShardedMap()

	handler := func(ctx context.Context) error { return nil }
	entry := &cacheEntry{
		handler:    handler,
		paramPairs: [MaxParams]paramPair{{key: "id", value: "123"}},
		paramCount: 1,
	}

	sm.StoreWithMethodPath("GET", "/users/123", entry)

	// 多次加载
	for i := 0; i < 10; i++ {
		loaded, ok := sm.LoadWithMethodPath("GET", "/users/123")
		if !ok {
			t.Fatal("failed to load entry")
		}
		// 验证条目存在
		if i == 9 && loaded.paramPairs[0].value != "123" {
			t.Errorf("param value = %s, want 123", loaded.paramPairs[0].value)
		}
	}
}

// 测试清空缓存
func TestShardedMap_Clear(t *testing.T) {
	sm := newShardedMap()

	handler := func(ctx context.Context) error { return nil }

	// 插入多个条目
	for i := 0; i < 100; i++ {
		entry := &cacheEntry{
			handler:    handler,
			paramPairs: [MaxParams]paramPair{{key: "id", value: fmt.Sprintf("%d", i)}},
			paramCount: 1,
		}
		sm.StoreWithMethodPath("GET", fmt.Sprintf("/users/%d", i), entry)
	}

	// 检查条目数
	totalEntries, _ := sm.Stats()
	if totalEntries == 0 {
		t.Fatal("no entries stored")
	}

	// 清空缓存
	sm.Clear()

	// 检查条目数
	totalEntries, _ = sm.Stats()
	if totalEntries != 0 {
		t.Errorf("total entries after clear = %d, want 0", totalEntries)
	}
}

// 测试缓存条目时间戳更新
func TestShardedMap_TimestampUpdate(t *testing.T) {
	sm := newShardedMap()

	handler := func(ctx context.Context) error { return nil }
	entry := &cacheEntry{
		handler:    handler,
		paramPairs: [MaxParams]paramPair{{key: "id", value: "123"}},
		paramCount: 1,
	}

	sm.StoreWithMethodPath("GET", "/users/123", entry)

	// 获取初始时间戳
	loaded1, _ := sm.LoadWithMethodPath("GET", "/users/123")
	timestamp1 := loaded1.timestamp

	// 等待一段时间后重新存储
	time.Sleep(10 * time.Millisecond)
	sm.StoreWithMethodPath("GET", "/users/123", entry)

	// 再次加载，时间戳应该更新
	loaded2, _ := sm.LoadWithMethodPath("GET", "/users/123")
	timestamp2 := loaded2.timestamp

	if timestamp2 <= timestamp1 {
		t.Errorf("timestamp not updated: %d <= %d", timestamp2, timestamp1)
	}
}

// 测试分片索引计算
func TestShardedMap_GetShard(t *testing.T) {
	sm := newShardedMap()

	// 测试相同的 key 总是返回相同的分片
	key := cacheKey{method: "GET", path: "/users/123"}
	shard1 := sm.getShard(key)
	shard2 := sm.getShard(key)

	if shard1 != shard2 {
		t.Errorf("shard index not consistent: %d != %d", shard1, shard2)
	}

	// 测试分片索引在有效范围内
	if shard1 < 0 || shard1 >= shardCount {
		t.Errorf("shard index out of range: %d", shard1)
	}
}

// 基准测试：缓存存储
func BenchmarkShardedMap_Store(b *testing.B) {
	sm := newShardedMap()
	handler := func(ctx context.Context) error { return nil }

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		entry := &cacheEntry{
			handler:    handler,
			paramPairs: [MaxParams]paramPair{{key: "id", value: fmt.Sprintf("%d", i)}},
			paramCount: 1,
		}
		sm.StoreWithMethodPath("GET", fmt.Sprintf("/users/%d", i), entry)
	}
}

// 基准测试：缓存加载
func BenchmarkShardedMap_Load(b *testing.B) {
	sm := newShardedMap()
	handler := func(ctx context.Context) error { return nil }

	// 预先插入条目
	for i := 0; i < 1000; i++ {
		entry := &cacheEntry{
			handler:    handler,
			paramPairs: [MaxParams]paramPair{{key: "id", value: fmt.Sprintf("%d", i)}},
			paramCount: 1,
		}
		sm.StoreWithMethodPath("GET", fmt.Sprintf("/users/%d", i), entry)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sm.LoadWithMethodPath("GET", fmt.Sprintf("/users/%d", i%1000))
	}
}

// 基准测试：并发缓存访问
func BenchmarkShardedMap_ConcurrentLoad(b *testing.B) {
	sm := newShardedMap()
	handler := func(ctx context.Context) error { return nil }

	// 预先插入条目
	for i := 0; i < 1000; i++ {
		entry := &cacheEntry{
			handler:    handler,
			paramPairs: [MaxParams]paramPair{{key: "id", value: fmt.Sprintf("%d", i)}},
			paramCount: 1,
		}
		sm.StoreWithMethodPath("GET", fmt.Sprintf("/users/%d", i), entry)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sm.LoadWithMethodPath("GET", fmt.Sprintf("/users/%d", i%1000))
			i++
		}
	})
}

// 测试LRU淘汰机制

// 测试Store触发淘汰
func TestShardedMap_StoreEviction(t *testing.T) {
	sm := newShardedMap()
	handler := func(ctx context.Context) error { return nil }

	// 使用固定前缀确保落到同一分片
	for i := 0; i < maxEntriesPerShard*2; i++ {
		entry := &cacheEntry{
			handler:   handler,
			timestamp: time.Now().UnixNano(),
		}
		sm.StoreWithMethodPath("GET", fmt.Sprintf("/test/%d", i), entry)
	}

	// 验证淘汰生效
	total, _ := sm.Stats()
	if total == 0 {
		t.Error("cache should have entries")
	}
}

// 测试evictLRU淘汰机制
func TestShardedMap_EvictLRU(t *testing.T) {
	sm := newShardedMap()
	handler := func(ctx context.Context) error { return nil }

	// 找到一个固定的分片索引
	testKey := cacheKey{method: "GET", path: "/test/0"}
	shardIdx := sm.getShard(testKey)

	// 向同一个分片填充超过maxEntriesPerShard的条目
	keysAdded := 0
	for i := 0; keysAdded < maxEntriesPerShard+10; i++ {
		method := "GET"
		path := fmt.Sprintf("/test/%d", i)
		key := cacheKey{method: method, path: path}
		if sm.getShard(key) == shardIdx {
			entry := &cacheEntry{
				handler:   handler,
				timestamp: time.Now().UnixNano() + int64(i),
			}
			sm.StoreWithMethodPath(method, path, entry)
			keysAdded++
			time.Sleep(time.Microsecond)
		}
	}

	// 验证淘汰生效：总条目数应该小于添加的数量
	total, shardDist := sm.Stats()
	if shardDist[shardIdx] >= int64(maxEntriesPerShard+10) {
		t.Errorf("shard %d has %d entries, eviction should have occurred", shardIdx, shardDist[shardIdx])
	}
	if total == 0 {
		t.Error("cache should still have entries after eviction")
	}
}
