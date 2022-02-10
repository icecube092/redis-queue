package redisq

import (
	"context"
	"github.com/go-redis/redis/v8"
)

type StringConverter interface {
	ToString() (string, error)
	FromString(s string) error
}

type Queue interface {
	// BeginRead locks queue for Scan.
	// Must be called before Scan.
	BeginRead(ctx context.Context) error
	// Push calls StringConverter.ToString and adds result to the queue.
	// Non-blocking.
	Push(ctx context.Context, t StringConverter) error
	// Scan gets next queue element and put it into StringConverter.FromString.
	// Element remain in queue.
	Scan(ctx context.Context, t StringConverter) error
	// Commit removes elements from queue and unlocks it.
	// Must call after sequence of Scan.
	Commit(ctx context.Context) error
	// Rollback move element into end of queue and unlocks it.
	// Must call after sequence of Scan.
	Rollback(ctx context.Context) error
	// Cancel breaks transaction. Elements remains in queue.
	// Must call after sequence of Scan.
	Cancel(ctx context.Context) error
}

// BeginMode defines behavior of Queue.BeginRead.
type BeginMode int64

const (
	// BlockBeginMode defines waiting while queue unlocks.
	BlockBeginMode BeginMode = iota
	// ErrBeginMode defines error if queue locked on Queue.BeginRead.
	ErrBeginMode
)

type QueueConfig struct {
	Conn redis.UniversalClient
	// Name is the name of queue.
	// Don't use same name in some queues.
	Name string
	// Typ is the type of queue
	Typ       StringConverter
	BeginMode BeginMode
}
