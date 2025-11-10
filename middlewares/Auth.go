package middlewares

import (
	"NewStudent/auth"
	"NewStudent/controllers"
	"NewStudent/dao"
	"NewStudent/models"
	"github.com/gin-gonic/gin"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if !strings.HasPrefix(authz, "Bearer") {
			controllers.ReturnError(c, 4001, "没有授权token")
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer"))
		claims, err := auth.Parse(tokenStr)
		if err != nil {
			controllers.ReturnError(c, 4001, "token已不可用")
			c.Abort()
			return
		}

		var u models.User
		err = dao.Db.Select("id, token_version").
			Where("id = ?", claims.UID).
			First(&u).Error
		if err != nil {
			controllers.ReturnError(c, 4001, "该用户不存在")
			c.Abort()
			return
		}

		if claims.Ver != u.TokenVersion {
			controllers.ReturnError(c, 4001, "令牌已失效")
			c.Abort()
			return
		}

		c.Set("uid", claims.UID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
