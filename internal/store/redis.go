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

type LeakyBucketResult struct {
	Allowed bool
	Depth   float64
}

var leakyBucketScript = redis.NewScript(`
local key      = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate     = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])

local bucket    = redis.call('HMGET', key, 'depth', 'last_time')
local depth     = tonumber(bucket[1]) or 0
local last_time = tonumber(bucket[2]) or now

local elapsed = now - last_time
local leaked  = elapsed * rate
local new_depth = math.max(0, depth - leaked)

if new_depth >= capacity then
    return {0, new_depth}
end

new_depth = new_depth + 1
redis.call('HMSET', key, 'depth', new_depth, 'last_time', now)
redis.call('EXPIRE', key, 3600)
return {1, new_depth}
`)

func (s *RedisStore) CheckLeakyBucket(
	ctx context.Context,
	key string,
	capacity float64,
	rate float64,
	now float64,
) (*LeakyBucketResult, error) {
	result, err := leakyBucketScript.Run(
		ctx, s.client,
		[]string{key},
		capacity, rate, now,
	).Slice()
	if err != nil {
		return nil, err
	}

	return &LeakyBucketResult{
		Allowed: result[0].(int64) == 1,
		Depth:   result[1].(float64),
	}, nil
}

type FixedWindowResult struct {
	Allowed bool
	Count   int64
}

var fixedWindowScript = redis.NewScript(`
local key    = KEYS[1]
local limit  = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local count = redis.call('INCR', key)

if count == 1 then
    redis.call('EXPIRE', key, window)
end

if count > limit then
    return {0, count}
end

return {1, count}
`)

func (s *RedisStore) CheckFixedWindow(
	ctx context.Context,
	key string,
	limit int64,
	windowSecs int64,
) (*FixedWindowResult, error) {
	result, err := fixedWindowScript.Run(
		ctx, s.client,
		[]string{key},
		limit, windowSecs,
	).Slice()
	if err != nil {
		return nil, err
	}

	return &FixedWindowResult{
		Allowed: result[0].(int64) == 1,
		Count:   result[1].(int64),
	}, nil
}
