package staticcache

import (
    "github.com/YuHover/staticcache/lru"
    "github.com/YuHover/staticcache/singleflight"
    "sync"
)

// StaticCache provides static resource cache service.
type StaticCache struct {
    name      string
    limit     int64
    mainCache *syncLRU
    getter    LocalGetter
    picker    *ConsistentPicker
    throttle  *singleflight.SingleFlight
}

var caches = make(map[string]*StaticCache)

func NewStaticCache(name string, limit int64, getter LocalGetter) *StaticCache {
    if getter == nil {
        panic("must provide local getter")
    }

    if _, dup := caches[name]; dup {
        panic("duplicate registration of cache: " + name)
    }

    sc := &StaticCache{
        name:      name,
        getter:    getter,
        mainCache: &syncLRU{},
        limit:     limit,
        throttle:  &singleflight.SingleFlight{},
    }
    caches[name] = sc
    return sc
}

func GetCache(name string) *StaticCache {
    return caches[name]
}

func (sc *StaticCache) SetConsistentPicker(picker *ConsistentPicker) {
    sc.picker = picker
}

func (sc *StaticCache) Get(key string) (byteView, error) {
    if bv, hit := sc.mainCache.get(key); hit {
        return bv, nil
    }

    // single flight
    bv, err := sc.throttle.Throttle(key, func() (any, error) {
        if sc.picker != nil {
            if remote := sc.picker.PickServer(key); remote != nil {
                if bv, err := sc.getFromRemote(remote, key); err == nil {
                    return bv, nil
                }
            }
        }

        return sc.getLocally(key)
    })

    if err == nil {
        return bv.(byteView), nil
    }
    return byteView{}, err
}

func (sc *StaticCache) getFromRemote(remote CacheServer, key string) (byteView, error) {
    bs, err := remote.Get(sc.name, key)
    if err != nil {
        return byteView{}, err
    }
    return byteView{bs: bs}, nil
}

func (sc *StaticCache) getLocally(key string) (byteView, error) {
    bs, err := sc.getter.Get(key)
    if err != nil {
        return byteView{}, err
    }

    bv := byteView{bs: cloneBytes(bs)}
    sc.populateCache(key, bv)
    return bv, nil
}

func (sc *StaticCache) populateCache(key string, value byteView) {
    sc.mainCache.add(key, value)

    for sc.mainCache.bytes() > sc.limit {
        sc.mainCache.removeOldest()
    }
}

// syncLRU is a concurrent secure LRUCache
type syncLRU struct {
    mu     sync.RWMutex
    lru    *lru.LRUCache[string]
    nbytes int64
}

func (sl *syncLRU) add(key string, value byteView) {
    sl.mu.Lock()
    defer sl.mu.Unlock()

    if sl.lru == nil {
        sl.lru = lru.New[string](0, func(key string, value any) {
            sl.nbytes -= int64(len(key)) + int64(value.(byteView).Len())
        })
    }
    sl.lru.Add(key, value)
    sl.nbytes += int64(len(key)) + int64(value.Len())
}

func (sl *syncLRU) get(key string) (byteView, bool) {
    sl.mu.RLock()
    defer sl.mu.RUnlock()

    if sl.lru == nil {
        return byteView{}, false
    }

    if val, ok := sl.lru.Get(key); ok {
        return val.(byteView), ok
    }
    return byteView{}, false
}

func (sl *syncLRU) removeOldest() {
    sl.mu.Lock()
    defer sl.mu.Unlock()

    if sl.lru != nil {
        sl.lru.RemoveOldest()
    }
}

func (sl *syncLRU) bytes() int64 {
    sl.mu.RLock()
    defer sl.mu.RUnlock()

    return sl.nbytes
}

// LocalGetter is the final data source.
// When both the local cache and remote cache miss, trying to get data from LocalGetter.
type LocalGetter interface {
    Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
    return f(key)
}
