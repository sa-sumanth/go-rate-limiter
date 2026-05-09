package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sa-sumanth/go-rate-limiter/internal/store"
)

const (
	swCounterLimit  = 10 // max requests per window
	swCounterWindow = 60 // window size in seconds
)

func SlidingWindowCounter(s *store.RedisStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now().Unix()
		windowStart := now - (now % swCounterWindow)
		prevStart := windowStart - swCounterWindow
		ip := c.ClientIP()

		currKey := fmt.Sprintf("sw_counter:%s:%d", ip, windowStart)
		prevKey := fmt.Sprintf("sw_counter:%s:%d", ip, prevStart)

		result, err := s.CheckSlidingWindowCounter(
			c.Request.Context(),
			currKey,
			prevKey,
			swCounterLimit,
			float64(swCounterWindow),
			float64(now),
		)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": "rate limiter error"},
			)
			return
		}

		if !result.Allowed {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{
					"error":    "rate limit exceeded",
					"estimate": result.Estimate,
				},
			)
			return
		}

		c.Next()
	}
}
