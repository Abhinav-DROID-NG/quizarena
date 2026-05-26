package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("request logger panic recovered", zap.Any("panic", rec))
			}
		}()
		start := time.Now()
		c.Next()
		path := ""
		method := ""
		if c.Request != nil && c.Request.URL != nil {
			path = c.Request.URL.Path
			method = c.Request.Method
		}
		logger.Info("http_request",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}
