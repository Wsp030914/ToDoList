package router

import (
	"NewStudent/controllers"
	"NewStudent/middlewares"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	r := gin.Default()

	public := r.Group("/api/v1")
	{
		public.POST("/login", controllers.UserController{}.Login)
		public.POST("/register", controllers.UserController{}.Register)

	}

	protected := r.Group("/api/v1")
	protected.Use(middlewares.AuthMiddleware())
	{
		protected.POST("/logout", controllers.UserController{}.Logout)
		//protected.POST("/user/update", controllers.UserController{}.Update)
		protected.POST("/projects", controllers.ProjectController{}.Create)
		protected.GET("/projects", controllers.ProjectController{}.List)
		protected.GET("/projects/:id", controllers.ProjectController{}.Search)
		protected.DELETE("/projects/:id", controllers.ProjectController{}.Delete)
	}

	return r
}
