package router

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

func New(db Pinger) *gin.Engine {
	r := gin.New()
	r.GET("/health", healthHandler(db))
	return r
}

func healthHandler(db Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
