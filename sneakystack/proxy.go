package sneakystack

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// ServiceHandler handles AWS API requests for a specific service.
type ServiceHandler interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

// Config configures a sneakystack proxy.
type Config struct {
	Services []string // Service names to handle (e.g., "sso-admin", "organizations").
}

// Proxy is an HTTP reverse proxy that routes requests to local service
// handlers for gap services and forwards everything else to LocalStack.
type Proxy struct {
	store        Store
	downstream   *url.URL
	handlers     map[string]ServiceHandler
	reverseProxy *httputil.ReverseProxy
}

// NewProxy creates a sneakystack proxy that forwards to the given downstream URL.
func NewProxy(store Store, downstreamURL string) (*Proxy, error) {
	downstream, err := url.Parse(downstreamURL)
	if err != nil {
		return nil, fmt.Errorf("parse downstream URL: %w", err)
	}

	p := &Proxy{
		store:      store,
		downstream: downstream,
		handlers:   make(map[string]ServiceHandler),
		reverseProxy: &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = downstream.Scheme
				req.URL.Host = downstream.Host
				req.Host = downstream.Host
			},
		},
	}

	return p, nil
}

// RegisterHandler maps a service prefix (from X-Amz-Target) to a handler.
func (p *Proxy) RegisterHandler(servicePrefix string, handler ServiceHandler) {
	p.handlers[servicePrefix] = handler
}

// ServeHTTP routes requests based on the X-Amz-Target header. If the target
// matches a registered handler, the request is handled locally. Otherwise,
// it is forwarded to LocalStack.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		// Extract service prefix (e.g., "AWSOrganizations" from "AWSOrganizations.ListAccounts").
		if handler, ok := p.matchHandler(target); ok {
			handler.Handle(w, r)
			return
		}
	}

	// Forward to LocalStack.
	p.reverseProxy.ServeHTTP(w, r)
}

// matchHandler finds a registered handler whose prefix matches the X-Amz-Target.
func (p *Proxy) matchHandler(target string) (ServiceHandler, bool) {
	for prefix, handler := range p.handlers {
		if strings.HasPrefix(target, prefix) {
			return handler, true
		}
	}

	return nil, false
}
