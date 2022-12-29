package consistenthash

import (
    "hash/crc32"
    "sort"
    "strconv"
)

type Hash func(data []byte) uint32

type virtualNode struct {
    name string
    hash uint32
}

type ConsistentHash struct {
    hashFn   Hash
    replicas int
    vNodes   []virtualNode // sorted for fast binary search
    vMap     map[virtualNode]string
    keys     map[string]struct{}
}

func New(replicas int, fn Hash) *ConsistentHash {
    ch := &ConsistentHash{
        hashFn:   fn,
        replicas: replicas,
        vNodes:   make([]virtualNode, 0),
        vMap:     make(map[virtualNode]string),
        keys:     map[string]struct{}{},
    }

    if ch.hashFn == nil {
        ch.hashFn = crc32.ChecksumIEEE
    }

    return ch
}

func (ch *ConsistentHash) Add(key string) {
    if _, ok := ch.keys[key]; ok {
        return
    }

    ch.keys[key] = struct{}{}
    for i := 0; i < ch.replicas; i++ {
        vName := key + strconv.Itoa(i)
        vNode := virtualNode{
            name: vName,
            hash: ch.hashFn([]byte(vName)),
        }

        ch.vNodes = append(ch.vNodes, vNode)
        ch.vMap[vNode] = key
    }
    sort.Slice(ch.vNodes, func(i, j int) bool {
        return ch.vNodes[i].hash <= ch.vNodes[j].hash
    })
}

func (ch *ConsistentHash) Get(key string) (string, bool) {
    if len(ch.vNodes) == 0 {
        return "", false
    }

    h := ch.hashFn([]byte(key))
    n := len(ch.vNodes)
    idx := sort.Search(n, func(i int) bool { return ch.vNodes[i].hash >= h }) % n
    return ch.vMap[ch.vNodes[idx]], true
}

func (ch *ConsistentHash) Remove(key string) {
    if _, ok := ch.keys[key]; !ok {
        return
    }

    for i := 0; i < ch.replicas; i++ {
        vName := key + strconv.Itoa(i)
        h := ch.hashFn([]byte(vName))

        idx := sort.Search(len(ch.vNodes), func(i int) bool { return ch.vNodes[i].hash >= h })
        for ; idx < len(ch.vNodes) && ch.vNodes[idx].hash == h && ch.vNodes[idx].name != vName; idx++ {
        }

        if idx < len(ch.vNodes) && ch.vNodes[idx].hash == h {
            delete(ch.vMap, ch.vNodes[idx])
            ch.vNodes = append(ch.vNodes[:idx], ch.vNodes[idx+1:]...)
        }
    }
}
