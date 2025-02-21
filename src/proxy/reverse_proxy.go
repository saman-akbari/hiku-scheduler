package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

// ReverseProxy is an interface for an object that proxies the client HTTP
// request to a worker. The implementation is free to choose any
// protocol or network stack
type ReverseProxy interface {
	// ProxyRequest is a function that proxies the client HTTP request to a
	// worker. It takes the same parameters as an HTTP server handler along
	// with the worker as the first parameter. It is expected to send a
	// response to the client using the ResponseWriter object.
	ProxyRequest(workerURL url.URL, w http.ResponseWriter, r *http.Request)
}

type HTTPReverseProxy struct {
	proxyMap map[url.URL]*httputil.ReverseProxy
	mutex    sync.Mutex
}

func (p *HTTPReverseProxy) getReverseProxyForWorker(workerURL url.URL) *httputil.ReverseProxy {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	proxy := p.proxyMap[workerURL]
	if proxy == nil {
		proxy = httputil.NewSingleHostReverseProxy(&workerURL)
		p.proxyMap[workerURL] = proxy
	}

	return proxy
}

func (p *HTTPReverseProxy) ProxyRequest(workerURL url.URL, w http.ResponseWriter, r *http.Request) {
	proxy := p.getReverseProxyForWorker(workerURL)
	proxy.ServeHTTP(w, r)
}

func NewHTTPReverseProxy() ReverseProxy {
	return &HTTPReverseProxy{
		proxyMap: make(map[url.URL]*httputil.ReverseProxy),
		mutex:    sync.Mutex{},
	}
}
