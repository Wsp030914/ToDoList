package auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization: Bearer <token> 读取
		authz := c.GetHeader("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing token"})
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
		claims, err := Parse(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid or expired token"})
			return
		}
		// 注入上下文，业务可用
		c.Set("uid", claims.UID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
