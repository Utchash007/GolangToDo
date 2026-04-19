package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery catches any panic in downstream handlers, logs it with the request ID
// and stack trace via slog, and returns 500 in the unified error shape.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if val := recover(); val != nil {
				slog.Error("panic recovered",
					"panic", val,
					"stack", string(debug.Stack()),
					"request_id", RequestIDFromContext(c.Request.Context()),
				)

				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, gin.H{
						"code": "internal_error",
						"errors": []gin.H{
							{"message": "internal server error"},
						},
					})
				}
			}
		}()
		c.Next()
	}
}
