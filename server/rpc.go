package server

import (
    "fmt"
    "github.com/YuHover/staticcache"
    "net/rpc"
)

type RPCRequest struct {
    CacheName string
    Key       string
}

type RPCResponse []byte

type RPCServer struct {
    address string
    name    string
}

type RPCOption func(rs *RPCServer)

func WithRPCName(name string) RPCOption {
    return func(rs *RPCServer) {
        rs.name = name
    }
}

func NewRPCServer(address string, opts ...RPCOption) *RPCServer {
    rs := &RPCServer{address: address}
    for _, opt := range opts {
        opt(rs)
    }
    if rs.name == "" {
        rs.name = address
    }

    return rs
}

func (rs *RPCServer) GetName() string {
    return rs.name
}

func (rs *RPCServer) GetSchema() string {
    return "rpc"
}

func (rs *RPCServer) ServeRPC(in RPCRequest, out *RPCResponse) error {
    cacheName, key := in.CacheName, in.Key
    if cacheName == "" || key == "" {
        return fmt.Errorf("bad request, want static cache name and key")
    }

    static := staticcache.GetCache(cacheName)
    if static == nil {
        return fmt.Errorf("no such static cache: %s", cacheName)
    }

    bv, err := static.Get(key)
    if err != nil {
        return fmt.Errorf("internal error: %v", err)
    }

    *out = bv.ByteSlice()
    return nil
}

func (rs *RPCServer) Get(cacheName string, key string) ([]byte, error) {
    if cacheName == "" || key == "" {
        return nil, fmt.Errorf("want static cache name and key")
    }

    client, err := rpc.Dial("tcp", rs.address)
    if err != nil {
        return nil, fmt.Errorf("connects to an RPC server at %s error: %v", rs.address, err)
    }

    res := &RPCResponse{}
    err = client.Call(rs.name+".ServeRPC", RPCRequest{CacheName: cacheName, Key: key}, res)
    if err != nil {
        return nil, fmt.Errorf("remote server error: %v", err)
    }

    return *res, nil
}
