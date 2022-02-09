package redis_queue

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis/v8"
)

type queue struct {
	conn    redis.UniversalClient
	mux     sync.Mutex
	txState int64

	name string
	typ  Stringer
}

func NewQueue(cfg *QueueConfig) (Queue, error) {
	q := &queue{
		conn: cfg.Conn,
		name: cfg.Name,
		typ:  cfg.Typ,
	}

	return q, nil
}

func (q *queue) finish() {
	atomic.StoreInt64(&q.txState, 0)
	q.mux.Unlock()
}

func (q *queue) isTx() bool {
	return atomic.LoadInt64(&q.txState) == 1
}

var (
	ErrWrongType = errors.New("type doesn't match")
	ErrNoTx      = errors.New("no transaction in progress")
)

func (q *queue) Push(ctx context.Context, t Stringer) error {
	if reflect.TypeOf(t) != reflect.TypeOf(q.typ) {
		return ErrWrongType
	}

	s, err := t.ToString()
	if err != nil {
		return fmt.Errorf("stringer error: %w", err)
	}

	if err = q.conn.LPush(ctx, q.name, s).Err(); err != nil {
		return fmt.Errorf("conn.LPush: %w", err)
	}

	return nil
}

func (q *queue) Scan(ctx context.Context, t Stringer) error {
	q.mux.Lock()
	atomic.StoreInt64(&q.txState, 1)

	result, err := q.conn.LRange(ctx, q.name, -1, -1).Result()
	if err != nil {
		q.finish()
		return fmt.Errorf("conn.LRange: %w", err)
	}

	if len(result) == 0 {
		q.finish()
		return redis.Nil
	}

	if err = t.FromString(result[0]); err != nil {
		q.finish()
		return fmt.Errorf("stringer error: %w", err)
	}

	return nil
}

func (q *queue) Commit(ctx context.Context) error {
	if !q.isTx() {
		return ErrNoTx
	}
	defer q.finish()

	if err := q.conn.RPop(ctx, q.name).Err(); err != nil {
		return fmt.Errorf("conn.RPop: %w", err)
	}

	return nil
}

func (q *queue) Break(ctx context.Context) error {
	if !q.isTx() {
		return ErrNoTx
	}
	defer q.finish()

	if err := q.conn.RPopLPush(ctx, q.name, q.name).Err(); err != nil {
		return fmt.Errorf("conn.RPopLPush: %w", err)
	}

	return nil
}
