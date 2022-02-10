package redisq_test

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/suite"
	"os"
	"sync"
	"testing"

	"github.com/icecube092/redisq"
)

type test struct {
	suite.Suite

	ctx  context.Context
	conn redis.UniversalClient
	cfg  *redisq.QueueConfig
}

func TestRun(t *testing.T) {
	suite.Run(t, &test{})
}

func (t *test) SetupSuite() {
	t.ctx = context.Background()

	redisConn := redis.NewClient(
		&redis.Options{
			Network: os.Getenv("TEST_REDIS_QUEUE_NETWORK"),
			Addr:    os.Getenv("TEST_REDIS_QUEUE_ADDR"),
		},
	)
	t.Require().NoError(redisConn.Ping(t.ctx).Err())
	t.conn = redisConn
	t.cfg = &redisq.QueueConfig{
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
		testName   = "testName"
		testStruct = &testStringer{Name: testName}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
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
	t.Require().ErrorIs(err, redisq.ErrNoTx)

	err = q.Rollback(t.ctx)
	t.Require().ErrorIs(err, redisq.ErrNoTx)
}

func (t *test) TestBreak() {
	var (
		testName   = "testName"
		testStruct = &testStringer{Name: testName}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
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
		testName   = "testName"
		wg         = sync.WaitGroup{}
		testStruct = &testStringer{Name: testName}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
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
		testName   = "testName"
		wg         = sync.WaitGroup{}
		testStruct = &testStringer{Name: testName}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
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
		testName    = "testName"
		testName2   = "testName2"
		testName3   = "testName3"
		testStruct  = &testStringer{Name: testName}
		testStruct2 = &testStringer{Name: testName2}
		testStruct3 = &testStringer{Name: testName3}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)
	err = q.Push(t.ctx, testStruct2)
	t.Require().NoError(err)
	err = q.Push(t.ctx, testStruct3)

	q.BeginRead()
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	getStruct2 := &testStringer{}
	err = q.Scan(t.ctx, getStruct2)
	t.Require().NoError(err)
	t.Require().Equal(testStruct2, getStruct2)

	getStruct3 := &testStringer{}
	err = q.Scan(t.ctx, getStruct3)
	t.Require().NoError(err)
	t.Require().Equal(testStruct3, getStruct3)

	err = q.Commit(t.ctx)
	t.Require().NoError(err)
}

func (t *test) TestCancel() {
	var (
		testName    = "testName"
		testName2   = "testName2"
		testStruct  = &testStringer{Name: testName}
		testStruct2 = &testStringer{Name: testName2}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)
	err = q.Push(t.ctx, testStruct2)
	t.Require().NoError(err)

	q.BeginRead()
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	err = q.Cancel()
	t.Require().NoError(err)

	q.BeginRead()
	getStruct2 := &testStringer{}
	err = q.Scan(t.ctx, getStruct2)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct2)

	err = q.Commit(t.ctx)
	t.Require().NoError(err)
}

func (t *test) TestReadToEnd() {
	var (
		testName    = "testName"
		testName2   = "testName2"
		testStruct  = &testStringer{Name: testName}
		testStruct2 = &testStringer{Name: testName2}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)
	err = q.Push(t.ctx, testStruct2)
	t.Require().NoError(err)

	q.BeginRead()
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	getStruct2 := &testStringer{}
	err = q.Scan(t.ctx, getStruct2)
	t.Require().NoError(err)
	t.Require().Equal(testStruct2, getStruct2)

	getStruct3 := &testStringer{}
	err = q.Scan(t.ctx, getStruct3)
	t.Require().ErrorIs(err, redis.Nil)

	err = q.Commit(t.ctx)
	t.Require().NoError(err)
}

func (t *test) TestReadToEndWithConcurrentPush() {
	var (
		testName    = "testName"
		testName2   = "testName2"
		testName3   = "testName3"
		testStruct  = &testStringer{Name: testName}
		testStruct2 = &testStringer{Name: testName2}
		testStruct3 = &testStringer{Name: testName3}
	)

	q, err := redisq.NewSeqQueue(t.cfg)
	t.Require().NoError(err)

	err = q.Push(t.ctx, testStruct)
	t.Require().NoError(err)
	err = q.Push(t.ctx, testStruct2)
	t.Require().NoError(err)

	q.BeginRead()
	getStruct := &testStringer{}
	err = q.Scan(t.ctx, getStruct)
	t.Require().NoError(err)
	t.Require().Equal(testStruct, getStruct)

	getStruct2 := &testStringer{}
	err = q.Scan(t.ctx, getStruct2)
	t.Require().NoError(err)
	t.Require().Equal(testStruct2, getStruct2)

	err = q.Push(t.ctx, testStruct3)
	t.Require().NoError(err)

	getStruct3 := &testStringer{}
	err = q.Scan(t.ctx, getStruct3)
	t.Require().NoError(err)
	t.Require().Equal(testStruct3, getStruct3)

	err = q.Commit(t.ctx)
	t.Require().NoError(err)
}
