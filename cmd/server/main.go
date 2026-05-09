package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/sa-sumanth/go-rate-limiter/internal/middleware"
	"github.com/sa-sumanth/go-rate-limiter/internal/store"
)

type Server struct {
	router *gin.Engine
	rdb    *redis.Client
}

func NewServer(rdb *redis.Client) *Server {
	s := &Server{
		router: gin.Default(),
		rdb:    rdb,
	}
	return s
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) setupRoutes() {
	pprof.Register(s.router)

	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	s.router.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "metrics stub"})
	})

	redisStore := store.NewRedisStore(s.rdb)

	s.router.GET("/token_bucket", middleware.TokenBucket(redisStore), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "request allowed"})
	})
	s.router.GET("/leaky_bucket", middleware.LeakyBucket(redisStore), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "request allowed"})
	})
	s.router.GET("/fixed_window", middleware.FixedWindow(redisStore), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "request allowed"})
	})
	s.router.GET("/sliding_window_log", middleware.SlidingWindowLog(redisStore), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "request allowed"})
	})
	s.router.GET("/sliding_window_counter", middleware.SlidingWindowCounter(redisStore), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "request allowed"})
	})
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	ctx := context.Background()

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	port := getEnv("PORT", "8080")

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("could not connect to redis: %v", err)
	}

	s := NewServer(rdb)
	s.setupRoutes()

	if err := s.Run(":" + port); err != nil {
		log.Fatalf("unable to start server: %v", err)
	}
}
