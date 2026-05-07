package logger

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GinAccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		fields := append(WithHTTPContext(c),
			zap.Int("status", c.Writer.Status()),
			zap.Float64("latency_ms", float64(time.Since(start).Nanoseconds())/1e6),
		)

		msg := "request completed"
		switch {
		case len(c.Errors) > 0:
			msg = "request completed with errors"
			fields = append(fields, zap.String("errors", c.Errors.String()))
		case c.Writer.Status() >= http.StatusInternalServerError:
			msg = "request failed"
		}

		switch status := c.Writer.Status(); {
		case status >= http.StatusInternalServerError:
			Gin().Error(msg, fields...)
		case status >= http.StatusBadRequest:
			Gin().Warn(msg, fields...)
		default:
			Gin().Info(msg, fields...)
		}
	}
}

func GinRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		fields := append(WithHTTPContext(c),
			zap.Any("panic", recovered),
			zap.ByteString("stack", debug.Stack()),
		)
		Gin().Error("panic recovered", fields...)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
