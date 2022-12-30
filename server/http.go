package server

import (
    "fmt"
    "github.com/YuHover/staticcache"
    "io"
    "net/http"
    "net/url"
    "strings"
)

const defaultBasePath = "/_cache/"

type HTTPServer struct {
    address  string
    basePath string
    name     string
}

type HttpOption func(hs *HTTPServer)

func WithBasePath(basePath string) HttpOption {
    return func(hs *HTTPServer) {
        hs.basePath = basePath
    }
}

func WithHTTPName(name string) HttpOption {
    return func(hs *HTTPServer) {
        hs.name = name
    }
}

func NewHTTPServer(address string, opts ...HttpOption) *HTTPServer {
    hs := &HTTPServer{
        address:  address,
        basePath: defaultBasePath,
    }

    for _, opt := range opts {
        opt(hs)
    }
    if hs.name == "" {
        hs.name = hs.address + hs.basePath
    }

    return hs
}

func (hs *HTTPServer) GetName() string {
    return hs.name
}

func (hs *HTTPServer) GetSchema() string {
    return "http"
}

// ServeHTTP serving for http://address/basepath/cachename/key
func (hs *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // parse request.
    if !strings.HasPrefix(r.URL.Path, hs.basePath) {
        http.Error(w, "bad request, want basepath/cachename/key", http.StatusBadRequest)
        return
    }
    parts := strings.Split(r.URL.Path[len(hs.basePath):], "/")
    if len(parts) != 2 || parts[1] == "" {
        http.Error(w, "bad request, want basepath/cachename/key", http.StatusBadRequest)
        return
    }
    cacheName, key := parts[0], parts[1]

    // fetch the value for this static/key.
    static := staticcache.GetCache(cacheName)
    if static == nil {
        http.Error(w, "no such static cache: "+cacheName, http.StatusNotFound)
        return
    }

    bv, err := static.Get(key)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/octet-stream")
    w.Write(bv.ByteSlice())
}

func (hs *HTTPServer) Get(cacheName string, key string) ([]byte, error) {
    if cacheName == "" || key == "" {
        return nil, fmt.Errorf("want static cache name and key")
    }

    u, err := url.JoinPath(hs.address, hs.basePath, url.PathEscape(cacheName), url.PathEscape(key))
    if err != nil {
        return nil, fmt.Errorf("assemble URL: %v", err)
    }

    res, err := http.Get(u)
    if err != nil {
        return nil, fmt.Errorf("GET method: %v", err)
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("remote server error: %v", res.Status)
    }

    bs, err := io.ReadAll(res.Body)
    if err != nil {
        return nil, fmt.Errorf("reading response body: %v", err)
    }

    return bs, nil
}
