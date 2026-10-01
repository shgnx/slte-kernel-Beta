package lru

import (
	"strings"
	"time"
)

// ResetLRU 以新的容量与选项重建缓存，并保留旧缓存中的存活条目。
//
// 来自上游 mihomo（smart 组用它按运行时配置调整缓存规模）。
func ResetLRU[K comparable, V any](oldCache *LruCache[K, V], newSize int, options ...Option[K, V]) *LruCache[K, V] {
	newCache := New[K, V](append(options, WithSize[K, V](newSize))...)
	oldCache.CloneTo(newCache)
	return newCache
}

// FilterByKeyPrefix 返回所有键以 prefix 开头的有效条目。
//
// 键类型不是 string 的条目会被跳过。来自上游 mihomo（smart 组按域名前缀聚合统计）。
func (c *LruCache[K, V]) FilterByKeyPrefix(prefix string) map[string]V {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make(map[string]V)
	now := time.Now().Unix()

	for k, le := range c.cache {
		keyStr, ok := any(k).(string)
		if !ok {
			continue
		}

		if !strings.HasPrefix(keyStr, prefix) {
			continue
		}

		if !c.staleReturn && c.maxAge > 0 && le.Value.expires <= now {
			c.deleteElement(le)
			continue
		}

		e := le.Value
		result[keyStr] = e.value
	}

	return result
}

// RemoveByKeyPrefix 删除所有键以 prefix 开头的条目，返回删除数量。
//
// 来自上游 mihomo。
func (c *LruCache[K, V]) RemoveByKeyPrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var removed int
	var keysToRemove []K

	for k := range c.cache {
		keyStr, ok := any(k).(string)
		if !ok {
			continue
		}

		if strings.HasPrefix(keyStr, prefix) {
			keysToRemove = append(keysToRemove, k)
		}
	}

	for _, k := range keysToRemove {
		c.delete(k)
		removed++
	}

	return removed
}
