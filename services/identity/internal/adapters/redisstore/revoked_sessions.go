package redisstore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RevokedSessions is the list the API Gateway will check on every
// request. An entry only needs to live as long as the longest-lived
// access token: after that, every token for the session has expired on
// its own.
type RevokedSessions struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRevokedSessions(client *redis.Client, accessTokenTTL time.Duration) *RevokedSessions {
	return &RevokedSessions{client: client, ttl: accessTokenTTL}
}

func (r *RevokedSessions) Add(ctx context.Context, sessionID string) error {
	if err := r.client.Set(ctx, "identity:session:revoked:"+sessionID, "1", r.ttl).Err(); err != nil {
		return fmt.Errorf("redisstore: add revoked session: %w", err)
	}
	return nil
}
