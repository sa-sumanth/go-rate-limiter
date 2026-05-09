package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sa-sumanth/go-rate-limiter/internal/store"
)

const (
	tbCapacity   = 10.0 // max tokens
	tbRefillRate = 2.0  // tokens per second
	tbTTL        = 3600
)

func TokenBucket(s *store.RedisStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "token_bucket:" + c.ClientIP()
		now := float64(time.Now().UnixMilli()) / 1000.0

		result, err := s.CheckTokenBucket(
			c.Request.Context(),
			key,
			tbCapacity,
			tbRefillRate,
			now,
			tbTTL,
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
				gin.H{"error": "rate limit exceeded", "remaining": 0},
			)
			return
		}

		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%.0f", result.Remaining))
		c.Next()
	}
}
