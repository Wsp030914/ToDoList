package router

import (
	"NewStudent/controllers"
	"NewStudent/log"
	"NewStudent/middlewares"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Router() *gin.Engine {
	cfg, err := log.LoadZapConfig()
	if err != nil {
		panic(err)
	}
	logger := log.InitZap(cfg)

	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	r := gin.New()
	r.Use(middlewares.AccessLogMiddleware(), middlewares.RecoveryWithZap())
	public := r.Group("/api/v1")
	{
		公共.POST("/login", controllers.UserController{}.Login)

		公共.POST("/register", controllers.UserController{}.Register)

	}

	protected := r.Group("/api/v1")
	protected.Use(middlewares.AuthMiddleware())
	{
		protected.PATCH("/users/me", controllers.UserController{}.Update)

		protected.POST("/logout", controllers.UserController{}.Logout)

		
		protected.POST("/projects", controllers.ProjectController{}.Create)

		protected.GET("/projects", controllers.ProjectController{}.List)

		protected.GET("/projects/:id", controllers.ProjectController{}.Search)

		protected.DELETE("/projects/:id", controllers.ProjectController{}.Delete)

		protected.PATCH("/projects/:id", controllers.ProjectController{}.Update)

		protected.POST("/tasks", controllers.TaskController{}.Create)

		protected.GET("/tasks", controllers.TaskController{}.List)

		protected.DELETE("/tasks/:id", controllers.TaskController{}.Delete)

		protected.GET("/tasks/:id", controllers.TaskController{}.Search)

		protected.DELETE("/tasks", controllers.TaskController{}.StatusDelete)

		protected.PATCH("/tasks/:id", controllers.TaskController{}.Update)

	}

	return r
}
