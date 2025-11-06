package router

import (
	"NewStudent/auth"
	"NewStudent/controllers"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	r := gin.Default()

	// 公开接口：不需要 JWT
	public := r.Group("/api/v1")
	{
		public.POST("/user/login", controllers.UserController{}.Login)
		public.POST("/user/register", controllers.UserController{}.Register)
	}

	// 受保护接口：需要 JWT
	protected := r.Group("/api/v1")
	protected.Use(auth.AuthMiddleware())
	{

	}

	return r
}
