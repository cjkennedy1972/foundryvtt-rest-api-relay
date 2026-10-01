package helpers

import (
	"fmt"
	"sync/atomic"
	"time"
)

var requestSeq atomic.Uint64

// NewRequestID returns an id for a request the relay forwards to a Foundry client and then waits on.
//
// The id keys the pending-request map, so it must be unique per request. A bare millisecond timestamp
// was not: two requests of the same kind within one millisecond (any concurrent burst) shared an id,
// the later pending entry overwrote the earlier, and the earlier request was never answered; it sat
// until the 20 s timeout and returned 408. The counter makes ids unique within the process.
func NewRequestID(kind string) string {
	return fmt.Sprintf("%s_%d_%d", kind, time.Now().UnixMilli(), requestSeq.Add(1))
}
