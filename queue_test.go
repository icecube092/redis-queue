package redis_queue_test

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"

	redis_queue "github.com/icecube092/redis-queue"
)

type test struct {
	suite.Suite

	ctx  context.Context
	conn redis.UniversalClient
	cfg  *redis_queue.QueueConfig
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
	t.cfg = &redis_queue.QueueConfig{
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

	q, err := redis_queue.NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)

	q.BeginRead()

	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	err = q.Commit(t.ctx)
	t.Require().NoError(err)

	err = q.Commit(t.ctx)
	t.Require().ErrorIs(err, redis_queue.ErrNoTx)

	err = q.Rollback(t.ctx)
	t.Require().ErrorIs(err, redis_queue.ErrNoTx)
}

func (t *test) TestBreak() {
	var (
		testName   = "hello"
		testStruct = &testStringer{Name: testName}
	)

	q, err := redis_queue.NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)

	q.BeginRead()
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	err = q.Rollback(t.ctx)
	t.Require().NoError(err)

	getStruct = &testStringer{}
	q.BeginRead()
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

	q, err := redis_queue.NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, &testStringer{Name: testName})
	t.Require().NoError(err)

	q.BeginRead()
	wg.Add(2)
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)
	wg.Done()

	go func() {
		q.BeginRead()
		defer q.Rollback(t.ctx)

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

	q, err := redis_queue.NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, &testStringer{Name: testName})
	t.Require().NoError(err)

	wg.Add(2)
	q.BeginRead()
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)
	wg.Done()

	go func() {
		getStruct := &testStringer{}
		q.BeginRead()
		defer q.Rollback(t.ctx)

		err = q.Scan(t.ctx, getStruct)
		t.NoError(err)
		t.Equal(testStruct, getStruct)
		wg.Done()
	}()

	err = q.Rollback(t.ctx)
	t.Require().NoError(err)
	wg.Wait()
}

func (t *test) TestSequentialScan() {
	var (
		testName    = "hello"
		testName2   = "hello2"
		testStruct  = &testStringer{Name: testName}
		testStruct2 = &testStringer{Name: testName2}
	)

	q, err := redis_queue.NewQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)
	err = q.Push(t.ctx, testStruct2)
	t.Require().NoError(err)

	q.BeginRead()
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct2, getStruct)

	getStruct = &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	q.Commit(t.ctx)
}
