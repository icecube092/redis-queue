# Redisq

It's a queue-over-redis that provides possibility to works with data like with
message broker. This is not a highload story, if you need maximum performance
and persistence - use more usual brokers. Lib works
over [redis lib](https://github.com/go-redis/redis).

## Requirements

- Redis-server >= 6.2.0

## Warnings

- Queue behaviour on cluster is undefined. Now it's single-node.
- Use persistent mode of redis-server

## Features:

- `BeginRead` locks queue for reading
- `Push` into queue without blocking
- `Scan` from queue without removing
- `Commit` scanned elements
- `Rollback` transaction with moving elements into end of queue
- `Cancel` breaks transaction, remains elements in queue

On `BeginRead` queue locks and unlocks after `Commit`, `Rollback`, or `Cancel`.

For generic usage with any type lib provides `Stringer` interface

See `queue_test.go` for examples.

## Roadmap

- [ ] Clustered queue
- [ ] Queue with priority
- [ ] Non-blocking transactions

## Run tests

- Set `TEST_REDIS_QUEUE_NETWORK`
- Set `TEST_REDIS_QUEUE_ADDR`
- Run `go test ./...`
