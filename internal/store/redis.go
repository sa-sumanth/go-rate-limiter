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

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int64:
		return float64(val)
	case float64:
		return val
	}
	return 0
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
local ttl         = tonumber(ARGV[4])

local bucket      = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens      = tonumber(bucket[1]) or capacity
local last_refill = tonumber(bucket[2]) or now

local elapsed    = now - last_refill
local new_tokens = math.min(capacity, tokens + elapsed * refill_rate)

if new_tokens < 1 then
    return {0, new_tokens}
end

new_tokens = new_tokens - 1
redis.call('HMSET', key, 'tokens', new_tokens, 'last_refill', now)
redis.call('EXPIRE', key, ttl)
return {1, new_tokens}
`)

func (s *RedisStore) CheckTokenBucket(
	ctx context.Context,
	key string,
	capacity float64,
	refillRate float64,
	now float64,
	ttl int64,
) (*TokenBucketResult, error) {
	result, err := tokenBucketScript.Run(
		ctx, s.client,
		[]string{key},
		capacity, refillRate, now, ttl,
	).Slice()
	if err != nil {
		return nil, err
	}

	return &TokenBucketResult{
		Allowed:   result[0].(int64) == 1,
		Remaining: toFloat64(result[1]),
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
local ttl      = tonumber(ARGV[4])

local bucket    = redis.call('HMGET', key, 'depth', 'last_time')
local depth     = tonumber(bucket[1]) or 0
local last_time = tonumber(bucket[2]) or now

local elapsed   = now - last_time
local leaked    = elapsed * rate
local new_depth = math.max(0, depth - leaked)

-- reject if adding one more request would overflow
if new_depth >= capacity then
    return {0, new_depth}
end

new_depth = new_depth + 1
redis.call('HMSET', key, 'depth', new_depth, 'last_time', now)
redis.call('EXPIRE', key, ttl)
return {1, new_depth}
`)

func (s *RedisStore) CheckLeakyBucket(
	ctx context.Context,
	key string,
	capacity float64,
	rate float64,
	now float64,
	ttl int64,
) (*LeakyBucketResult, error) {
	result, err := leakyBucketScript.Run(
		ctx, s.client,
		[]string{key},
		capacity, rate, now, ttl,
	).Slice()
	if err != nil {
		return nil, err
	}

	return &LeakyBucketResult{
		Allowed: result[0].(int64) == 1,
		Depth:   toFloat64(result[1]),
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

type SlidingWindowResult struct {
	Allowed bool
	Count   int64
}

var slidingWindowLogScript = redis.NewScript(`
local key    = KEYS[1]
local limit  = tonumber(ARGV[1])
local now    = tonumber(ARGV[2])
local window = tonumber(ARGV[3])

local cutoff = now - window

redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)

local count = redis.call('ZCARD', key)

if count >= limit then
    return {0, count}
end

redis.call('ZADD', key, now, now)
redis.call('EXPIRE', key, window)
return {1, count + 1}
`)

func (s *RedisStore) CheckSlidingWindowLog(
	ctx context.Context,
	key string,
	limit int64,
	windowSecs float64,
	now float64,
) (*SlidingWindowResult, error) {
	result, err := slidingWindowLogScript.Run(
		ctx, s.client,
		[]string{key},
		limit, now, windowSecs,
	).Slice()
	if err != nil {
		return nil, err
	}

	return &SlidingWindowResult{
		Allowed: result[0].(int64) == 1,
		Count:   result[1].(int64),
	}, nil
}

type SlidingWindowCounterResult struct {
	Allowed  bool
	Estimate float64
}

var slidingWindowCounterScript = redis.NewScript(`
local curr_key = KEYS[1]
local prev_key = KEYS[2]
local limit    = tonumber(ARGV[1])
local now      = tonumber(ARGV[2])
local window   = tonumber(ARGV[3])

local curr_count = tonumber(redis.call('GET', curr_key)) or 0
local prev_count = tonumber(redis.call('GET', prev_key)) or 0

local elapsed_ratio = (now % window) / window
local estimate      = prev_count * (1 - elapsed_ratio) + curr_count

if estimate >= limit then
    return {0, estimate}
end

redis.call('INCR', curr_key)
redis.call('EXPIRE', curr_key, window * 2)
return {1, estimate + 1}
`)

func (s *RedisStore) CheckSlidingWindowCounter(
	ctx context.Context,
	currKey string,
	prevKey string,
	limit int64,
	windowSecs float64,
	now float64,
) (*SlidingWindowCounterResult, error) {
	result, err := slidingWindowCounterScript.Run(
		ctx, s.client,
		[]string{currKey, prevKey},
		limit, now, windowSecs,
	).Slice()
	if err != nil {
		return nil, err
	}

	return &SlidingWindowCounterResult{
		Allowed:  result[0].(int64) == 1,
		Estimate: toFloat64(result[1]),
	}, nil
}
