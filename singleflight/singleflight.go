package singleflight

import "sync"

type call struct {
    wg  sync.WaitGroup
    ret any
    err error
}

type SingleFlight struct {
    mu     sync.Mutex
    record map[string]*call
}

func (sf *SingleFlight) Throttle(key string, fn func() (any, error)) (any, error) {
    sf.mu.Lock()
    if sf.record == nil {
        sf.record = make(map[string]*call)
    }
    if c, ok := sf.record[key]; ok {
        sf.mu.Unlock()
        c.wg.Wait()
        return c.ret, c.err
    }

    c := &call{}
    c.wg.Add(1)
    sf.record[key] = c
    sf.mu.Unlock()

    c.ret, c.err = fn()
    c.wg.Done()

    sf.mu.Lock()
    delete(sf.record, key)
    sf.mu.Unlock()

    return c.ret, c.err
}
