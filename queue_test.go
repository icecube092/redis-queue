package redis_queue

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/suite"
)

type test struct {
	suite.Suite

	ctx  context.Context
	conn redis.UniversalClient
	cfg  *QueueConfig
}

func TestRun(t *testing.T) {
	suite.Run(t, &test{})
}

func (t *test) SetupSuite() {
	t.ctx = context.Background()

	redisConn := redis.NewClient(
		&redis.Options{
			Network: "tcp",
			Addr:    "127.0.0.1:6379",
		},
	)
	t.Require().NoError(redisConn.Ping(t.ctx).Err())
	t.conn = redisConn
	t.cfg = &QueueConfig{
		Conn: t.conn,
		Name: "test",
		Typ:  &testStringer{},
	}
}

func (t *test) TearDownTest() {
	t.conn.FlushAll(t.ctx)
}

type testStringer struct {
	Name string
}

func (t *testStringer) ToString() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}

	return string(b), nil
}

func (t *testStringer) FromString(s string) error {
	err := json.Unmarshal([]byte(s), t)
	if err != nil {
		panic(err)
	}

	return nil
}

func (t *test) TestOk() {
	var (
		testName   = "hello"
		testStruct = &testStringer{Name: testName}
	)

	q, err := NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)

	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	err = q.Commit(t.ctx)
	t.Require().NoError(err)

	err = q.Commit(t.ctx)
	t.Require().ErrorIs(err, ErrNoTx)

	err = q.Break(t.ctx)
	t.Require().ErrorIs(err, ErrNoTx)
}

func (t *test) TestBreak() {
	var (
		testName   = "hello"
		testStruct = &testStringer{Name: testName}
	)

	q, err := NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)

	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	err = q.Break(t.ctx)
	t.Require().NoError(err)

	getStruct = &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)
}

func (t *test) TestParallelGet() {
	var (
		testName   = "hello"
		wg         = sync.WaitGroup{}
		testStruct = &testStringer{Name: testName}
	)

	q, err := NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, &testStringer{Name: testName})
	t.Require().NoError(err)

	wg.Add(2)
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)
	wg.Done()

	go func() {
		getStruct := &testStringer{}
		err = q.Scan(t.ctx, getStruct)
		t.ErrorIs(err, redis.Nil)
		wg.Done()
	}()

	err = q.Commit(t.ctx)
	t.Require().NoError(err)
	wg.Wait()
}

func (t *test) TestParallelGetAfterBreak() {
	var (
		testName   = "hello"
		wg         = sync.WaitGroup{}
		testStruct = &testStringer{Name: testName}
	)

	q, err := NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, &testStringer{Name: testName})
	t.Require().NoError(err)

	wg.Add(2)
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)
	wg.Done()

	go func() {
		getStruct := &testStringer{}
		err = q.Scan(t.ctx, getStruct)
		t.NoError(err)
		t.Equal(testStruct, getStruct)
		wg.Done()
	}()

	err = q.Break(t.ctx)
	t.Require().NoError(err)
	wg.Wait()
}
