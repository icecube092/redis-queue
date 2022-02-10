package redis_queue

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go.uber.org/atomic"
	"sync"
)

type queue struct {
	conn      redis.UniversalClient
	mux       sync.Mutex
	txState   atomic.Int64
	txCounter atomic.Int64

	name string
	typ  Stringer
}

func validateConfig(cfg *QueueConfig) error {
	if cfg.Conn == nil {
		return fmt.Errorf("nil conn")
	}
	if err := cfg.Conn.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	if cfg.Name == "" {
		return fmt.Errorf("empty queue name")
	}
	if cfg.Typ == nil {
		return fmt.Errorf("empty type")
	}

	return nil
}

func (q *queue) finish() {
	q.txCounter.Store(0)
	q.txState.Store(0)
	q.mux.Unlock()
}

func (q *queue) txExists() bool {
	return q.txState.Load() == 1
}
