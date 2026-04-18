package middleware

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey struct{}

// RequestIDFromContext returns the request ID stored in ctx, or empty string if not set.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

// RequestID generates a UUID v4 per request, injects it into context, and sets X-Request-ID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.NewRandom()
		if err != nil {
			slog.Warn("failed to generate request ID", "error", err)
			c.Next()
			return
		}

		requestID := id.String()
		c.Set("requestID", requestID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), contextKey{}, requestID))
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
