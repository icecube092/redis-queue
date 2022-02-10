package redisq

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go.uber.org/atomic"
	"reflect"
	"sync"
)

type queue struct {
	conn      redis.UniversalClient
	mux       sync.Mutex
	txState   atomic.Int64
	txCounter atomic.Int64

	startFunc func() error

	name string
	typ  StringConverter
}

func NewSeqQueue(cfg *QueueConfig) (Queue, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("validateConfig: %w", err)
	}

	q := &queue{
		conn: cfg.Conn,
		name: cfg.Name,
		typ:  cfg.Typ,
	}

	if err := q.setStartMode(cfg.BeginMode); err != nil {
		return nil, fmt.Errorf("setStartMode: %w", err)
	}

	return q, nil
}

var (
	ErrWrongType = errors.New("type doesn't match")
	ErrNoTx      = errors.New("no transaction in progress")
	ErrAlreadyTx = errors.New("transaction already opened")
)

func (q *queue) Push(ctx context.Context, t StringConverter) error {
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

func (q *queue) BeginRead(ctx context.Context) error {
	return q.startFunc()
}

func (q *queue) Scan(ctx context.Context, t StringConverter) error {
	if !q.txExists() {
		return ErrNoTx
	}

	position := -(q.txCounter.Load() + 1)
	result, err := q.conn.LRange(ctx, q.name, position, position).Result()
	if err != nil {
		return fmt.Errorf("conn.LRange: %w", err)
	}

	if len(result) == 0 {
		return redis.Nil
	}

	if err = t.FromString(result[0]); err != nil {
		return fmt.Errorf("stringer error: %w", err)
	}

	q.txCounter.Inc()

	return nil
}

func (q *queue) Commit(ctx context.Context) error {
	if !q.txExists() {
		return ErrNoTx
	}
	defer q.finish()

	if err := q.conn.RPopCount(ctx, q.name, int(q.txCounter.Load())).Err(); err != nil {
		return fmt.Errorf("conn.RPopCount: %w", err)
	}

	return nil
}

func (q *queue) Rollback(ctx context.Context) error {
	if !q.txExists() {
		return ErrNoTx
	}
	defer q.finish()

	pipe := q.conn.Pipeline()
	defer pipe.Discard()
	for i := int64(0); i < q.txCounter.Load(); i++ {
		if err := pipe.RPopLPush(ctx, q.name, q.name).Err(); err != nil {
			return fmt.Errorf("conn.RPopLPush: %w", err)
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("pipeline.Exec: %w", err)
	}

	return nil
}

func (q *queue) Cancel(ctx context.Context) error {
	if !q.txExists() {
		return ErrNoTx
	}
	defer q.finish()

	return nil
}
