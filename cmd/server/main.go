package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-contrib/pprof"
	"github.com/sa-sumanth/go-rate-limiter/internal/middleware"
	"github.com/sa-sumanth/go-rate-limiter/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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

func main() {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("could not connect to redis: %v", err)
	}

	s := NewServer(rdb)
	s.setupRoutes()

	if err := s.Run(":8080"); err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}
}
