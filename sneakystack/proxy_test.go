package sneakystack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockHandler struct {
	called bool
}

func (m *mockHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	m.called = true
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"handled": true}`))
}

func TestProxy_RoutesToHandler(t *testing.T) {
	t.Parallel()

	// Create a downstream server.
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"downstream": true}`))
	}))
	defer downstream.Close()

	store := NewMapStore()
	proxy, err := NewProxy(store, downstream.URL)
	if err != nil {
		t.Fatalf("NewProxy() error = %v", err)
	}

	handler := &mockHandler{}
	proxy.RegisterHandler("AWSOrganizations", handler)

	// Request with matching X-Amz-Target should go to handler.
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", http.NoBody)
	req.Header.Set("X-Amz-Target", "AWSOrganizations.ListAccounts")
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if !handler.called {
		t.Error("handler was not called for matching X-Amz-Target")
	}
}

func TestProxy_ForwardsToDownstream(t *testing.T) {
	t.Parallel()

	downstreamCalled := false
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		downstreamCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"downstream": true}`))
	}))
	defer downstream.Close()

	store := NewMapStore()
	proxy, err := NewProxy(store, downstream.URL)
	if err != nil {
		t.Fatalf("NewProxy() error = %v", err)
	}

	// Request without X-Amz-Target should go to downstream.
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/_localstack/health", http.NoBody)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if !downstreamCalled {
		t.Error("downstream was not called for non-matching request")
	}
}

func TestProxy_UnmatchedTargetForwards(t *testing.T) {
	t.Parallel()

	downstreamCalled := false
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		downstreamCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	store := NewMapStore()
	proxy, err := NewProxy(store, downstream.URL)
	if err != nil {
		t.Fatalf("NewProxy() error = %v", err)
	}

	proxy.RegisterHandler("AWSOrganizations", &mockHandler{})

	// Request with non-matching X-Amz-Target should forward.
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", http.NoBody)
	req.Header.Set("X-Amz-Target", "AWSCloudFormation.CreateStack")
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if !downstreamCalled {
		t.Error("downstream was not called for unmatched X-Amz-Target")
	}
}
