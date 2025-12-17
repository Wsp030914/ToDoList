package middlewares

import (
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strings"
)

func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		lg := utils.CtxLogger(c)
		authz := c.GetHeader("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			utils.ReturnError(c, 4001, "没有授权token")
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
		claims, err := utils.Parse(tokenStr)
		if err != nil {
			utils.ReturnError(c, 4001, "token已不可用")
			c.Abort()
			return
		}

		err = authService.ValidateJti(c.Request.Context(), lg, claims.RegisteredClaims.ID)
		if err != nil {
			var ae *service.AppError
			if errors.As(err, &ae) {
				utils.ReturnError(c, ae.Code, ae.Message)
			} else {
				lg.Error("auth_Validate_Jti", zap.Error(err))
				utils.ReturnError(c, 5001, "服务忙，请稍后重试")
			}
			c.Abort()
			return
		}

		err = authService.ValidateVersion(c.Request.Context(), lg, claims.UID, claims.Ver)
		if err != nil {
			var ae *service.AppError
			if errors.As(err, &ae) {
				utils.ReturnError(c, ae.Code, ae.Message)
			} else {
				lg.Error("auth_Validate_version", zap.Error(err))
				utils.ReturnError(c, 5001, "服务忙，请稍后重试")
			}
			c.Abort()
			return
		}

		c.Set("uid", claims.UID)
		c.Set("username", claims.Username)
		c.Set("claims", claims)
		c.Next()
	}
}
