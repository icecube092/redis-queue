# Redis-queue

It's a queue-over-redis that provides possibility to works with data like with
message broker. This is not a highload story, if you need maximum performance
and persistence - use more usual brokers.
Lib works over `github.com/go-redis/redis`.

## Features:

- `Push` into queue without blocking
- `Scan` from queue without removing
- `Commit` last scanned element
- `Break` transaction with moving last element into end of queue

On `Scan` queue locks and unlocks after `Commit` or `Break`.

For generic usage with any type lib provides `Stringer` interface

See `queue_test.go` for examples.