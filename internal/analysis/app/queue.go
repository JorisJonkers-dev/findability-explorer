package app

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// queue holds the ids of the analyses waiting to run, first in first out, without a bound: a
// scan an administrator starts always waits its turn and is never refused. The database holds
// each analysis itself, so the queue holds only ids.
type queue struct {
	mu  sync.Mutex
	ids []uuid.UUID
	// ready wakes one waiting worker; it holds at most one wake-up.
	ready chan struct{}
}

func newQueue() *queue { return &queue{ready: make(chan struct{}, 1)} }

// push adds id at the back and wakes a waiting worker. It never blocks.
func (q *queue) push(id uuid.UUID) {
	q.mu.Lock()
	q.ids = append(q.ids, id)
	q.mu.Unlock()
	q.wake()
}

// pop takes the id at the front, waiting until there is one or ctx ends; ok is false when ctx
// ended first.
func (q *queue) pop(ctx context.Context) (id uuid.UUID, ok bool) {
	for {
		q.mu.Lock()
		if len(q.ids) > 0 {
			id = q.ids[0]
			q.ids = q.ids[1:]
			more := len(q.ids) > 0
			q.mu.Unlock()
			if more {
				q.wake() // another worker may take the next one
			}
			return id, true
		}
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return uuid.Nil, false
		case <-q.ready:
		}
	}
}

func (q *queue) wake() {
	select {
	case q.ready <- struct{}{}:
	default:
	}
}
