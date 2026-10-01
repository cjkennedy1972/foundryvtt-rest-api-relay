package helpers

import (
	"strings"
	"sync"
	"testing"
)

func TestNewRequestIDIsUniqueUnderConcurrency(t *testing.T) {
	const workers, perWorker = 32, 2000
	ids := make(chan string, workers*perWorker)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				ids <- NewRequestID("execute-js")
			}
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[string]struct{}, workers*perWorker)
	for id := range ids {
		if !strings.HasPrefix(id, "execute-js_") {
			t.Fatalf("id must keep its kind prefix, got %q", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate request id %q: concurrent requests would overwrite each other's pending entry", id)
		}
		seen[id] = struct{}{}
	}
}
