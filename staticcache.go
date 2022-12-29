package staticcache

import (
    "github.com/YuHover/staticcache/lru"
    "github.com/YuHover/staticcache/singleflight"
    "sync"
)

type LocalGetter interface {
    Get(key string) ([]byte, error)
}

// A GetterFunc implements LocalGetter with a function.
type GetterFunc func(key string) ([]byte, error)

// Get implements LocalGetter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
    return f(key)
}

type Group struct {
    name      string
    limit     int64
    mainCache *cache
    getter    LocalGetter
    picker    Picker
    throttle  *singleflight.SingleFlight
}

var (
    mu     sync.RWMutex
    groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, limit int64, getter LocalGetter) *Group {
    if getter == nil {
        panic("nil LocalGetter")
    }

    mu.Lock()
    defer mu.Unlock()

    if _, dup := groups[name]; dup {
        panic("duplicate registration of group " + name)
    }

    g := &Group{
        name:      name,
        getter:    getter,
        mainCache: &cache{},
        limit:     limit,
        throttle:  &singleflight.SingleFlight{},
    }
    groups[name] = g
    return g
}

func (g *Group) RegisterPicker(picker Picker) {
    g.picker = picker
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
    mu.RLock()
    defer mu.RUnlock()

    return groups[name]
}

func (g *Group) Get(key string) (byteView, error) {
    if bv, hit := g.mainCache.get(key); hit {
        return bv, nil
    }

    // single flight
    bv, err := g.throttle.Throttle(key, func() (any, error) {
        if g.picker != nil {
            if remote := g.picker.PickServer(key); remote != nil {
                if bv, err := g.getFromRemote(remote, key); err == nil {
                    return bv, nil
                }
            }
        }

        return g.getLocally(key)
    })

    if err == nil {
        return bv.(byteView), nil
    }
    return byteView{}, err
}

func (g *Group) getFromRemote(remote RemoteServer, key string) (byteView, error) {
    bs, err := remote.Get(g.name, key)
    if err != nil {
        return byteView{}, err
    }
    return byteView{bs: bs}, nil
}

func (g *Group) getLocally(key string) (byteView, error) {
    bs, err := g.getter.Get(key)
    if err != nil {
        return byteView{}, err
    }

    bv := byteView{bs: cloneBytes(bs)}
    g.populateCache(key, bv)
    return bv, nil
}

func (g *Group) populateCache(key string, value byteView) {
    g.mainCache.add(key, value)

    for g.mainCache.bytes() > g.limit {
        g.mainCache.removeOldest()
    }
}

// cache is concurrent secure LRUCache
type cache struct {
    mu     sync.RWMutex
    lru    *lru.LRUCache[string]
    nbytes int64
}

func (c *cache) add(key string, value byteView) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.lru == nil {
        c.lru = lru.New[string](0, func(key string, value any) {
            c.nbytes -= int64(len(key)) + int64(value.(byteView).Len())
        })
    }
    c.lru.Add(key, value)
    c.nbytes += int64(len(key)) + int64(value.Len())
}

func (c *cache) get(key string) (byteView, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if c.lru == nil {
        return byteView{}, false
    }

    if val, ok := c.lru.Get(key); ok {
        return val.(byteView), ok
    }
    return byteView{}, false
}

func (c *cache) removeOldest() {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.lru != nil {
        c.lru.RemoveOldest()
    }
}

func (c *cache) bytes() int64 {
    c.mu.RLock()
    defer c.mu.RUnlock()

    return c.nbytes
}
