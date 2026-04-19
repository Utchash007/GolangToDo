package router

import (
	"context"
	"net/http"

	"GolangToDo/internal/middleware"
	"GolangToDo/internal/task"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type Pinger interface {
	PingContext(ctx context.Context) error
}

func New(db *sqlx.DB) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.GET("/health", HealthHandler(db))

	taskHandler := task.NewHandler(task.NewService(db))
	taskHandler.RegisterRoutes(r)

	return r
}

// HealthHandler returns a Gin handler that pings db and reports service health.
// Exported so tests can use it directly with a mock Pinger.
func HealthHandler(db Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
