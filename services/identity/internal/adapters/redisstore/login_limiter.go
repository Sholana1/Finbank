// Package redisstore keeps the short-lived security state that lives in Redis.
package redisstore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// recordFailure runs inside Redis as one atomic step:
//  1. add one to the failure counter;
//  2. on the first failure, start the counting window;
//  3. at the limit, set the lock and clear the counter.
//
// Doing these as separate commands from Go would leave gaps: a crash
// between INCR and PEXPIRE would leave a counter that never expires.
var recordFailure = redis.NewScript(`
local failures = redis.call('INCR', KEYS[1])
if failures == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if failures >= tonumber(ARGV[2]) then
    redis.call('SET', KEYS[2], '1', 'PX', ARGV[3])
    redis.call('DEL', KEYS[1])
end
return failures
`)

type LoginLimiter struct {
	client       *redis.Client
	maxFailures  int
	window       time.Duration
	lockDuration time.Duration
}

func NewLoginLimiter(client *redis.Client, maxFailures int, window, lockDuration time.Duration) *LoginLimiter {
	return &LoginLimiter{
		client:       client,
		maxFailures:  maxFailures,
		window:       window,
		lockDuration: lockDuration,
	}
}

func (l *LoginLimiter) IsLocked(ctx context.Context, key string) (bool, error) {
	n, err := l.client.Exists(ctx, lockKey(key)).Result()
	if err != nil {
		return false, fmt.Errorf("redisstore: check lock: %w", err)
	}
	return n > 0, nil
}

func (l *LoginLimiter) RecordFailure(ctx context.Context, key string) error {
	err := recordFailure.Run(ctx, l.client,
		[]string{failuresKey(key), lockKey(key)},
		l.window.Milliseconds(), l.maxFailures, l.lockDuration.Milliseconds(),
	).Err()
	if err != nil {
		return fmt.Errorf("redisstore: record failure: %w", err)
	}
	return nil
}

func (l *LoginLimiter) Reset(ctx context.Context, key string) error {
	if err := l.client.Del(ctx, failuresKey(key)).Err(); err != nil {
		return fmt.Errorf("redisstore: reset failures: %w", err)
	}
	return nil
}

// The braces are a Redis Cluster hash tag: both keys for one subject
// land on the same node, which a script touching both requires.
func failuresKey(key string) string { return "identity:login:{" + key + "}:failures" }
func lockKey(key string) string     { return "identity:login:{" + key + "}:lock" }
