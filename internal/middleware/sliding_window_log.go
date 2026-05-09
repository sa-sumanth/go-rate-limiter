package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sa-sumanth/go-rate-limiter/internal/store"
)

const (
	swLogLimit  = 10   // max requests per window
	swLogWindow = 60.0 // window size in seconds
)

func SlidingWindowLog(s *store.RedisStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "sliding_window_log:" + c.ClientIP()
		now := float64(time.Now().UnixMilli()) / 1000.0

		result, err := s.CheckSlidingWindowLog(
			c.Request.Context(),
			key,
			swLogLimit,
			swLogWindow,
			now,
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
				gin.H{"error": "rate limit exceeded", "count": result.Count},
			)
			return
		}

		c.Next()
	}
}
