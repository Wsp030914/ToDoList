package middlewares

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"github.com/google/uuid"
	"time"
)

func AccessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		reqID := c.GetString("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		// 为本次请求准备一个带字段的 logger，挂到 context
		lg := zap.L().With(
			zap.String("request_id", reqID),
			zap.String("client_ip", c.ClientIP()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
		)
		c.Set("logger", lg)
		c.Header("X-Request-ID", reqID)

		c.Next()

		// 收尾：记录耗时/状态码/错误
		latency := time.Since(start)
		status := c.Writer.Status()
		lg.Info("access",
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.Int("size", c.Writer.Size()),
		)
	}
}

func RecoveryWithZap() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// 拿请求级 logger（或退回全局）
				lg, _ := c.Get("logger")
				logger, _ := lg.(*zap.Logger)
				if logger == nil {
					logger = zap.L()
				}

				logger.Error("panic recovered",
					zap.Any("panic", rec),
					zap.Stack("stack"),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 5000, "msg": "Internal Server Error"})
			}
		}()
		c.Next()
	}
}
