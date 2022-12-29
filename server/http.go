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

// HTTPServer implements RemoteServer
type HTTPServer struct {
    baseURL  string
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

// NewHTTPServer initializes an HTTP server
func NewHTTPServer(baseURL string, opts ...HttpOption) *HTTPServer {
    hs := &HTTPServer{
        baseURL:  baseURL,
        basePath: defaultBasePath,
    }

    for _, opt := range opts {
        opt(hs)
    }
    if hs.name == "" {
        hs.name = hs.baseURL + hs.basePath
    }

    return hs
}

func (hs *HTTPServer) GetName() string {
    return hs.name
}

// ServeHTTP serving for base/basePath/groupname/key
func (hs *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Parse request.
    if !strings.HasPrefix(r.URL.Path, hs.basePath) {
        http.Error(w, "bad request, want basepath/groupname/key", http.StatusBadRequest)
        return
    }
    parts := strings.Split(r.URL.Path[len(hs.basePath):], "/")
    if len(parts) != 2 || parts[1] == "" {
        http.Error(w, "bad request, want basepath/groupname/key", http.StatusBadRequest)
        return
    }
    groupName, key := parts[0], parts[1]

    // Fetch the value for this group/key.
    group := staticcache.GetGroup(groupName)
    if group == nil {
        http.Error(w, "no such group: "+groupName, http.StatusNotFound)
        return
    }

    bv, err := group.Get(key)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/octet-stream")
    w.Write(bv.ByteSlice())
}

// Get will use GET request data in remote server
func (hs *HTTPServer) Get(groupName string, key string) ([]byte, error) {
    if groupName == "" || key == "" {
        return nil, fmt.Errorf("want group name and key")
    }

    u, err := url.JoinPath(hs.baseURL, hs.basePath, url.PathEscape(groupName), url.PathEscape(key))
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
