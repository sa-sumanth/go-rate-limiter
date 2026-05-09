package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sa-sumanth/go-rate-limiter/internal/store"
)

const (
	fwLimit  = 10 // max requests per window
	fwWindow = 60 // window size in seconds
)

func FixedWindow(s *store.RedisStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now().Unix()
		windowStart := now - (now % fwWindow)
		key := fmt.Sprintf(
			"fixed_window:%s:%d",
			c.ClientIP(),
			windowStart,
		)

		result, err := s.CheckFixedWindow(
			c.Request.Context(),
			key,
			fwLimit,
			fwWindow,
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
