# Redisq

Redisq is a queue-over-redis that provides simple way to works with queues
stored in Redis.

Lib works over [redis lib](https://github.com/go-redis/redis).

### Advantages

- Simple - just launch redis-server on persistent mode
- Enough persistence
- Good for periodic poll model
- No need for mature message broker on simple queue task, use if you already
  have redis-server
- Distributed write

### Disadvantages

- No broadcasting
- No distributed read
- One transaction per time

### Warnings

- Use `noeviction` mode of redis-server for persist your data

## Requirements

- Redis-server >= 6.2.0
- Go 1.17+

## Installation

`go get github.com/icecube092/redisq@latest`

## Features:

- `BeginRead` locks queue for reading
- `Push` into queue without blocking
- `Scan` from queue without removing
- `Commit` scanned elements
- `Rollback` transaction with moving elements into end of queue
- `Cancel` breaks transaction, remains elements in queue

On `BeginRead` queue locks for read and unlocks after `Commit`, `Rollback`,
or `Cancel`.

For generic usage with any type lib provides `StringConverter` interface

## Example

```go
type intStringer int // implements redisq.StringConverter

setter := intStringer(1)

q, _ := redisq.NewSeqQueue(cfg)

q.Push(ctx, setter)

getter := intStringer(0)

q.BeginRead(ctx)
q.Scan(ctx, getter)
q.Commit(ctx)

// getter will be intStringer(1)
```

See `queue_test.go` for more examples.

## Roadmap

- [ ] Distributed read
- [ ] Queue with priority
- [ ] Non-blocking transactions

## Run tests

**Warning:** tests clear the storage after every test.

- Set `TEST_REDIS_QUEUE_NETWORK` and `TEST_REDIS_QUEUE_ADDR`, or just launch
  redis-server on `localhost:6379`
- Run `go test ./...`
