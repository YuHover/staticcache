package server

import (
    "fmt"
    "github.com/YuHover/staticcache"
    "net/rpc"
)

type RPCRequest struct {
    GroupName string
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

func (rs *RPCServer) ServeRPC(in RPCRequest, out *RPCResponse) error {
    groupName, key := in.GroupName, in.Key
    if groupName == "" || key == "" {
        return fmt.Errorf("bad request, want group name and key")
    }

    group := staticcache.GetGroup(groupName)
    if group == nil {
        return fmt.Errorf("no such group: %s", groupName)
    }

    bv, err := group.Get(key)
    if err != nil {
        return fmt.Errorf("internal error: %v", err)
    }

    *out = bv.ByteSlice()
    return nil
}

func (rs *RPCServer) GetName() string {
    return rs.name
}

func (rs *RPCServer) Get(groupName string, key string) ([]byte, error) {
    if groupName == "" || key == "" {
        return nil, fmt.Errorf("want group name and key")
    }

    client, err := rpc.Dial("tcp", rs.address)
    if err != nil {
        return nil, fmt.Errorf("connects to an RPC server at %s error: %v", rs.address, err)
    }

    res := &RPCResponse{}
    err = client.Call(rs.name+".ServeRPC", RPCRequest{GroupName: groupName, Key: key}, res)
    if err != nil {
        return nil, fmt.Errorf("remote server error: %v", err)
    }

    return *res, nil
}
