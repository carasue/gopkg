package lru

import (
	"github.com/jxskiss/gopkg/fasthash"
	"reflect"
	"time"
)

// NewMultiCache returns a hash-shared lru cache instance which is suitable
// to use for heavy lock contention use-case. It keeps same interface with
// the lru cache instance returned by NewCache function.
// Generally NewCache should be used instead of this unless you are sure that
// you are facing the lock contention problem.
func NewMultiCache(buckets, bucketCapacity int) *multiCache {
	mc := &multiCache{
		buckets: uintptr(buckets),
		cache:   make([]*cache, buckets),
	}
	for i := 0; i < buckets; i++ {
		mc.cache[i] = NewCache(bucketCapacity)
	}
	return mc
}

type multiCache struct {
	buckets uintptr
	cache   []*cache
}

func (c *multiCache) Len() (n int) {
	for _, c := range c.cache {
		n += c.Len()
	}
	return
}

func (c *multiCache) Get(key interface{}) (v interface{}, exists, expired bool) {
	h := fasthash.Hash(key)
	return c.cache[h%c.buckets].Get(key)
}

func (c *multiCache) GetQuiet(key interface{}) (v interface{}, exists, expired bool) {
	h := fasthash.Hash(key)
	return c.cache[h%c.buckets].GetQuiet(key)
}

func (c *multiCache) GetNotStale(key interface{}) (v interface{}, exists bool) {
	h := fasthash.Hash(key)
	return c.cache[h%c.buckets].GetNotStale(key)
}

func (c *multiCache) MGetInt64(keys []int64) map[int64]interface{} {
	grpKeys := c.groupInt64Keys(keys)

	var res map[int64]interface{}
	for idx, keys := range grpKeys {
		grp := c.cache[idx].MGetInt64(keys)
		if res == nil {
			res = grp
		} else {
			for k, v := range grp {
				res[k] = v
			}
		}
	}
	return res
}

func (c *multiCache) MGetString(keys []string) map[string]interface{} {
	grpKeys := c.groupStringKeys(keys)

	var res map[string]interface{}
	for idx, keys := range grpKeys {
		grp := c.cache[idx].MGetString(keys)
		if res == nil {
			res = grp
		} else {
			for k, v := range grp {
				res[k] = v
			}
		}
	}
	return res
}

func (c *multiCache) Set(key, value interface{}, ttl time.Duration) {
	h := fasthash.Hash(key)
	c.cache[h%c.buckets].Set(key, value, ttl)
}

func (c *multiCache) MSet(kvmap interface{}, ttl time.Duration) {
	m := reflect.ValueOf(kvmap)
	keys := m.MapKeys()

	for _, key := range keys {
		value := m.MapIndex(key)
		c.Set(key.Interface(), value.Interface(), ttl)
	}
}

func (c *multiCache) Del(key interface{}) {
	h := fasthash.Hash(key)
	c.cache[h%c.buckets].Del(key)
}

func (c *multiCache) MDelInt64(keys []int64) {
	grpKeys := c.groupInt64Keys(keys)

	for idx, keys := range grpKeys {
		c.cache[idx].MDelInt64(keys)
	}
}

func (c *multiCache) MDelString(keys []string) {
	grpKeys := c.groupStringKeys(keys)

	for idx, keys := range grpKeys {
		c.cache[idx].MDelString(keys)
	}
}

func (c *multiCache) groupInt64Keys(keys []int64) map[uintptr][]int64 {
	grpKeys := make(map[uintptr][]int64)
	for _, key := range keys {
		idx := fasthash.Int64(key) % c.buckets
		grpKeys[idx] = append(grpKeys[idx], key)
	}
	return grpKeys
}

func (c *multiCache) groupStringKeys(keys []string) map[uintptr][]string {
	grpKeys := make(map[uintptr][]string)
	for _, key := range keys {
		idx := fasthash.String(key) % c.buckets
		grpKeys[idx] = append(grpKeys[idx], key)
	}
	return grpKeys
}
