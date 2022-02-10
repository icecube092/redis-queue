package redis_queue

import (
	"context"
	"github.com/go-redis/redis/v8"
)

type Stringer interface {
	ToString() (string, error)
	FromString(s string) error
}

type Queue interface {
	// BeginRead locks queue for Scan.
	// Must be called before Scan.
	BeginRead()
	// Push calls Stringer.ToString and adds result to the queue.
	// Non-blocking.
	Push(ctx context.Context, t Stringer) error
	// Scan gets next queue element and put it into Stringer.FromString.
	// Element remain in queue.
	Scan(ctx context.Context, t Stringer) error
	// Commit removes elements from queue and unlock it.
	// Must call after sequence of Scan.
	Commit(ctx context.Context) error
	// Rollback move element into end of queue and unlock it.
	// Must call after sequence of Scan.
	Rollback(ctx context.Context) error
	// Cancel breaks transaction. Elements remains in queue.
	// Must call after sequence of Scan.
	Cancel() error
}

type QueueConfig struct {
	Conn redis.UniversalClient
	Name string
	Typ  Stringer
}
