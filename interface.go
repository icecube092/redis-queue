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
	// Push calls Stringer.ToString and adds result to the queue. Non-blocking.
	Push(ctx context.Context, t Stringer) error
	// Scan gets next queue element and put it into t.FromString.
	// Element left in queue. Blocking.
	Scan(ctx context.Context, t Stringer) error
	// Commit removes element from queue and unlock it.
	Commit(ctx context.Context) error
	// Break move element into end of queue and unlock it.
	Break(ctx context.Context) error
}

type QueueConfig struct {
	Conn redis.UniversalClient
	Name string
	Typ  Stringer
}
