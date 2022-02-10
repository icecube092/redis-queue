package redisq

import (
	"context"
	"fmt"
)

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

func (q *queue) setStartMode(beginMode BeginMode) error {
	switch beginMode {
	case ErrBeginMode:
		q.startFunc = q.beginErr
	case BlockBeginMode:
		q.startFunc = q.beginBlock
	default:
		return fmt.Errorf("unknown BeginMode")
	}

	return nil
}

func (q *queue) beginErr() error {
	if q.txExists() {
		return ErrAlreadyTx
	}

	return q.beginBlock()
}

func (q *queue) beginBlock() error {
	q.mux.Lock()
	q.txState.Store(1)
	q.txCounter.Store(0)

	return nil
}

func (q *queue) txExists() bool {
	return q.txState.Load() == 1
}
