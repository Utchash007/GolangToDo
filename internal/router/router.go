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
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.GET("/health", healthHandler(db))

	taskHandler := task.NewHandler(task.NewService(db))
	taskHandler.RegisterRoutes(r)

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
