package middleware

import (
	"server/logger"
	"time"

	"github.com/gin-gonic/gin"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method

		l := logger.Ctx(c)

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				// 错误时也保留 context，方便排查
				l.Errorw(e,
					"status", status,
					"method", method,
					"path", path,
					"query", query,
					"latency_ms", float64(latency.Microseconds())/1000.0,
					"user_agent", c.Request.UserAgent(),
				)
			}
		} else {
			l.Infow("HTTP Request",
				"status", status,
				"method", method,
				"path", path,
				"query", query,
				"latency_ms", float64(latency.Microseconds())/1000.0,
				"user_agent", c.Request.UserAgent(),
			)
		}
	}
}

