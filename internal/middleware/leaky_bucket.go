package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sa-sumanth/go-rate-limiter/internal/store"
)

const (
	lbCapacity = 10.0 // max queue depth
	lbRate     = 2.0  // requests leaked per second
	lbTTL      = 3600
)

func LeakyBucket(s *store.RedisStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "leaky_bucket:" + c.ClientIP()
		now := float64(time.Now().UnixMilli()) / 1000.0

		result, err := s.CheckLeakyBucket(
			c.Request.Context(),
			key,
			lbCapacity,
			lbRate,
			now,
			lbTTL,
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
				gin.H{"error": "rate limit exceeded", "depth": result.Depth},
			)
			return
		}

		c.Next()
	}
}
