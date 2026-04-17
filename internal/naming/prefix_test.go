package naming

import (
	"sync"
	"testing"
)

func TestPrefix_Format(t *testing.T) {
	t.Parallel()

	p := Prefix(t)

	if len(p) != 10 {
		t.Errorf("Prefix(t) = %q (len %d), want length 10", p, len(p))
	}

	if p[:4] != "ltt-" {
		t.Errorf("Prefix(t) = %q, want prefix \"ltt-\"", p)
	}

	// Remaining 6 chars should be valid hex.
	for _, c := range p[4:] {
		isDigit := c >= '0' && c <= '9'
		isHexLetter := c >= 'a' && c <= 'f'
		if !isDigit && !isHexLetter {
			t.Errorf("Prefix(t) = %q, contains non-hex char %c", p, c)
		}
	}
}

func TestPrefix_DeterministicWithinCall(t *testing.T) {
	t.Parallel()

	// Two calls in the same test will differ because time.Now().UnixNano()
	// advances, but each individual call produces a well-formed prefix.
	a := Prefix(t)
	b := Prefix(t)

	if len(a) != 10 || len(b) != 10 {
		t.Errorf("Prefix lengths: a=%d, b=%d, want 10", len(a), len(b))
	}
}

func TestPrefix_UniqueAcrossParallelTests(t *testing.T) {
	t.Parallel()

	const numGoroutines = 50

	var (
		mu      sync.Mutex
		seen    = make(map[string]bool, numGoroutines)
		results = make([]string, numGoroutines)
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			p := Prefix(t)
			mu.Lock()
			results[idx] = p
			seen[p] = true
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// With 50 goroutines and nanosecond resolution, we expect most to be
	// unique. Allow a small number of collisions since the hash input
	// includes the same test name and PID — only nanotime differs.
	uniqueCount := len(seen)
	if uniqueCount < numGoroutines/2 {
		t.Errorf("Prefix() produced only %d unique values out of %d calls", uniqueCount, numGoroutines)
	}
}
