package staticcache

import (
    "github.com/YuHover/staticcache/consistenthash"
    "log"
)

type CacheServer interface {
    Get(cacheName string, key string) ([]byte, error)
    GetName() string
    GetSchema() string
}

const defaultReplicas = 50

// ConsistentPicker implements picker using consistent hash algorithm to organize and pick servers.
type ConsistentPicker struct {
    replicas    int
    hashFn      consistenthash.Hash
    hashCircle  *consistenthash.ConsistentHash
    servers     map[string]CacheServer
    localServer CacheServer
}

type ConsistentPickerOption func(cp *ConsistentPicker)

func WithReplicas(replicas int) ConsistentPickerOption {
    return func(cp *ConsistentPicker) {
        cp.replicas = replicas
    }
}

func WithHashFn(hashFn func(data []byte) uint32) ConsistentPickerOption {
    return func(cp *ConsistentPicker) {
        cp.hashFn = hashFn
    }
}

func NewConsistentPicker(opts ...ConsistentPickerOption) *ConsistentPicker {
    cp := &ConsistentPicker{replicas: defaultReplicas}
    for _, opt := range opts {
        opt(cp)
    }

    return cp
}

func (cs *ConsistentPicker) SetServers(local CacheServer, remotes ...CacheServer) {
    if local == nil {
        panic("must provide local server")
    }
    cs.localServer = local
    cs.hashCircle = consistenthash.New(cs.replicas, cs.hashFn)
    cs.servers = make(map[string]CacheServer)

    for _, serv := range append(remotes, local) {
        name := serv.GetName()
        cs.hashCircle.Add(name)
        if _, ok := cs.servers[name]; ok {
            panic("duplicated server: " + name)
        }
        cs.servers[name] = serv
    }
}

func (cs *ConsistentPicker) PickServer(key string) CacheServer {
    if servName, ok := cs.hashCircle.Get(key); ok && servName != cs.localServer.GetName() {
        picked := cs.servers[servName]
        log.Printf("[%s][%s] is picked.\n", picked.GetSchema(), servName)
        return picked
    }
    return nil
}
