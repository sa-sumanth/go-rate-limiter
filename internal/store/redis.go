package store

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// ── Token Bucket ─────────────────────────────────────────────────────────────

type TokenBucketResult struct {
	Allowed   bool
	Remaining float64
}

var tokenBucketScript = redis.NewScript(`
local key         = KEYS[1]
local capacity    = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now         = tonumber(ARGV[3])

local bucket      = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens      = tonumber(bucket[1]) or capacity
local last_refill = tonumber(bucket[2]) or now

local elapsed   = now - last_refill
local new_tokens = math.min(capacity, tokens + elapsed * refill_rate)

if new_tokens < 1 then
    return {0, new_tokens}
end

new_tokens = new_tokens - 1
redis.call('HMSET', key, 'tokens', new_tokens, 'last_refill', now)
redis.call('EXPIRE', key, 3600)
return {1, new_tokens}
`)

func (s *RedisStore) CheckTokenBucket(
	ctx context.Context,
	key string,
	capacity float64,
	refillRate float64,
	now float64,
) (*TokenBucketResult, error) {
	result, err := tokenBucketScript.Run(
		ctx, s.client,
		[]string{key},
		capacity, refillRate, now,
	).Slice()
	if err != nil {
		return nil, err
	}

	return &TokenBucketResult{
		Allowed:   result[0].(int64) == 1,
		Remaining: result[1].(float64),
	}, nil
}
