package sneakystack

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestMapStore_CRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMapStore()

	// Put.
	err := store.Put(ctx, "widget", "w1", map[string]string{"name": "Widget One"})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Get.
	obj, err := store.Get(ctx, "widget", "w1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	widget := obj.(map[string]string)
	if widget["name"] != "Widget One" {
		t.Errorf("Get() name = %q, want Widget One", widget["name"])
	}

	// List.
	items, err := store.List(ctx, "widget", Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() count = %d, want 1", len(items))
	}

	// Delete.
	err = store.Delete(ctx, "widget", "w1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Get after delete should fail.
	_, err = store.Get(ctx, "widget", "w1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete = %v, want ErrNotFound", err)
	}
}

func TestMapStore_GetNotFound(t *testing.T) {
	t.Parallel()

	store := NewMapStore()

	_, err := store.Get(context.Background(), "widget", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(nonexistent) = %v, want ErrNotFound", err)
	}
}

func TestMapStore_DeleteNotFound(t *testing.T) {
	t.Parallel()

	store := NewMapStore()

	err := store.Delete(context.Background(), "widget", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete(nonexistent) = %v, want ErrNotFound", err)
	}
}

func TestMapStore_ListEmpty(t *testing.T) {
	t.Parallel()

	store := NewMapStore()

	items, err := store.List(context.Background(), "widget", Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if items != nil {
		t.Errorf("List(empty) = %v, want nil", items)
	}
}

func TestMapStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMapStore()

	const numOps = 100
	var wg sync.WaitGroup
	wg.Add(numOps * 3)

	// Concurrent puts.
	for i := range numOps {
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("w%d", idx)
			_ = store.Put(ctx, "widget", id, map[string]string{"idx": id})
		}(i)
	}

	// Concurrent reads.
	for range numOps {
		go func() {
			defer wg.Done()
			_, _ = store.List(ctx, "widget", Filter{})
		}()
	}

	// Concurrent deletes (may fail — that's ok).
	for i := range numOps {
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("w%d", idx)
			_ = store.Delete(ctx, "widget", id)
		}(i)
	}

	wg.Wait()
}
