package lru

import "container/list"

type LRUCache[K comparable] struct {
    // maxEntries is the maximum number of cache entries before
    // an item is evicted. Zero means no limit.
    maxEntries int

    // onEvicted optionally specifies a callback function to be
    // executed when an entry is purged from the cache.
    onEvicted func(key K, value any)

    ll    *list.List
    cache map[K]*list.Element
}

type entry[K comparable] struct {
    key   K
    value any
}

func New[K comparable](maxEntries int, onEvicted func(K, any)) *LRUCache[K] {
    return &LRUCache[K]{
        maxEntries: maxEntries,
        onEvicted:  onEvicted,
        ll:         list.New(),
        cache:      make(map[K]*list.Element),
    }
}

func (c *LRUCache[K]) Get(key K) (any, bool) {
    if elem, ok := c.cache[key]; ok {
        c.ll.MoveToFront(elem)
        return elem.Value.(*entry[K]).value, true
    }
    return nil, false
}

func (c *LRUCache[K]) Add(key K, value any) {
    if elem, ok := c.cache[key]; ok {
        c.ll.MoveToFront(elem)
        elem.Value.(*entry[K]).value = value
        return
    }

    c.cache[key] = c.ll.PushFront(&entry[K]{key, value})
    if c.maxEntries != 0 && c.ll.Len() > c.maxEntries {
        c.RemoveOldest()
    }
}

func (c *LRUCache[K]) RemoveOldest() {
    if back := c.ll.Back(); back != nil {
        c.removeElement(back)
    }
}

func (c *LRUCache[K]) Remove(key K) {
    if elem, ok := c.cache[key]; ok {
        c.removeElement(elem)
    }
}

func (c *LRUCache[K]) removeElement(elem *list.Element) {
    c.ll.Remove(elem)
    kv := elem.Value.(*entry[K])
    delete(c.cache, kv.key)
    if c.onEvicted != nil {
        c.onEvicted(kv.key, kv.value)
    }
}

func (c *LRUCache[K]) Len() int {
    return c.ll.Len()
}
