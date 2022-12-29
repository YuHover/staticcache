package staticcache

import (
    "github.com/YuHover/staticcache/consistenthash"
)

type RemoteServer interface {
    Get(groupName string, key string) ([]byte, error)
    GetName() string
}

type Picker interface {
    // SetServers set remote servers to accept query when the local cache misses.
    SetServers(local RemoteServer, remotes ...RemoteServer)
    // PickServer pick a remote server to which the query will be sent.
    // The pick strategy must ensure there will not be non-stop picking,
    // e.g. A picks A, A picks A... or A picks B, B picks A...
    PickServer(key string) RemoteServer
}

const defaultReplicas = 50

type ConsistentPicker struct {
    replicas    int
    hashFn      consistenthash.Hash
    hashCircle  *consistenthash.ConsistentHash
    servers     map[string]RemoteServer
    localServer RemoteServer
}

type CpOption func(cp *ConsistentPicker)

func WithReplicas(replicas int) CpOption {
    return func(cp *ConsistentPicker) {
        cp.replicas = replicas
    }
}

func WithHashFn(hashFn func(data []byte) uint32) CpOption {
    return func(cp *ConsistentPicker) {
        cp.hashFn = hashFn
    }
}

func NewConsistentPicker(opts ...CpOption) *ConsistentPicker {
    cp := &ConsistentPicker{replicas: defaultReplicas}
    for _, opt := range opts {
        opt(cp)
    }

    return cp
}

func (cs *ConsistentPicker) SetServers(local RemoteServer, remotes ...RemoteServer) {
    if local == nil {
        panic("must provide local server")
    }
    cs.localServer = local
    cs.hashCircle = consistenthash.New(cs.replicas, cs.hashFn)
    cs.servers = make(map[string]RemoteServer)

    for _, serv := range append(remotes, local) {
        name := serv.GetName()
        cs.hashCircle.Add(name)
        if _, ok := cs.servers[name]; ok {
            panic("duplicated server " + name)
        }
        cs.servers[name] = serv
    }
}

func (cs *ConsistentPicker) PickServer(key string) RemoteServer {
    if servName, ok := cs.hashCircle.Get(key); ok && servName != cs.localServer.GetName() {
        return cs.servers[servName]
    }
    return nil
}
