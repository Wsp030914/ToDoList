package log

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CtxLogger(c *gin.Context) *zap.Logger {
	if v, ok := c.Get("logger"); ok {
		if lg, ok := v.(*zap.Logger); ok && lg != nil {
			return lg
		}
	}
	// 兜底用全局 logger（需要在 main 初始化 zap 并 ReplaceGlobals）
	return zap.L()
}
