// Package sneakystack provides a LocalStack gap-filling HTTP proxy with an in-memory store.
package sneakystack

import (
	"context"
	"fmt"
	"sync"
)

// Store is the persistence abstraction for service handlers. Each service
// gets its own typed wrapper built on top of this.
type Store interface {
	Put(ctx context.Context, kind, id string, obj any) error
	Get(ctx context.Context, kind, id string) (any, error)
	List(ctx context.Context, kind string, filter Filter) ([]any, error)
	Delete(ctx context.Context, kind, id string) error
}

// Filter constrains List results.
type Filter struct {
	Parent string            // E.g. instance ARN, OU id.
	Tags   map[string]string // Optional tag match.
}

// ErrNotFound indicates the requested resource was not found.
var ErrNotFound = fmt.Errorf("resource not found")

// MapStore is a Store backed by plain Go maps protected by sync.RWMutex.
type MapStore struct {
	mu   sync.RWMutex
	data map[string]map[string]any // kind -> id -> obj
}

// NewMapStore creates an empty MapStore.
func NewMapStore() *MapStore {
	return &MapStore{
		data: make(map[string]map[string]any),
	}
}

// Put stores an object.
func (s *MapStore) Put(_ context.Context, kind, id string, obj any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[kind] == nil {
		s.data[kind] = make(map[string]any)
	}

	s.data[kind][id] = obj

	return nil
}

// Get retrieves an object by kind and id.
func (s *MapStore) Get(_ context.Context, kind, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, ok := s.data[kind]
	if !ok {
		return nil, ErrNotFound
	}

	obj, ok := bucket[id]
	if !ok {
		return nil, ErrNotFound
	}

	return obj, nil
}

// List returns all objects of a given kind, optionally filtered.
func (s *MapStore) List(_ context.Context, kind string, _ Filter) ([]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bucket, ok := s.data[kind]
	if !ok {
		return nil, nil
	}

	result := make([]any, 0, len(bucket))
	for _, obj := range bucket {
		result = append(result, obj)
	}

	return result, nil
}

// Delete removes an object.
func (s *MapStore) Delete(_ context.Context, kind, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, ok := s.data[kind]
	if !ok {
		return ErrNotFound
	}

	if _, ok := bucket[id]; !ok {
		return ErrNotFound
	}

	delete(bucket, id)

	return nil
}

// Ensure MapStore implements Store.
var _ Store = (*MapStore)(nil)
