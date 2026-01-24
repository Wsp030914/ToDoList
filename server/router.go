package main

import (
	"ToDoList/server/async"
	"ToDoList/server/config"
	"ToDoList/server/handler"
	"ToDoList/server/middlewares"
	"ToDoList/server/service"
	"context"

	"github.com/redis/go-redis/v9"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

type App struct {
	Bus *async.EventBus
	Rdb *redis.Client
	Db  *gorm.DB
}

func NewRouter(ctx context.Context, app *App) *gin.Engine {
	cfg, err := config.LoadZapConfig()
	if err != nil {
		panic(err)
	}
	logger := config.InitZap(cfg)

	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	r := gin.New()
	r.Use(middlewares.AccessLogMiddleware(), middlewares.RecoveryWithZap())
	r.Use(middlewares.RateLimitMiddleware(5, 10))
	r.Use(middlewares.CORSMiddleware())
	

	userSvc := service.NewUserService(app.Bus)
	userCtl := handler.NewUserHandler(userSvc)
	projectSvc := service.NewProjectService(app.Bus)
	projectCtl := handler.NewProjectHandler(projectSvc)
	taskSvc := service.NewTaskService(app.Bus)
	taskCtl := handler.NewTaskHandler(taskSvc)
	authSvc := service.NewAuthService(app.Bus)
	public := r.Group("/api/v1")
	{
		public.POST("/login", userCtl.Login)
		public.POST("/register", userCtl.Register)
	}

	protected := r.Group("/api/v1")
	protected.Use(middlewares.AuthMiddleware(authSvc))
	{
		protected.PATCH("/users/me", userCtl.Update)
		protected.POST("/logout", userCtl.Logout)
		protected.GET("/projects/:id", projectCtl.GetProjectByID)
		protected.GET("/projects", projectCtl.Search)
		protected.POST("/projects", projectCtl.Create)
		protected.PATCH("/projects/:id", projectCtl.Update)
		protected.DELETE("/projects/:id", projectCtl.Delete)

		protected.POST("/tasks", taskCtl.Create)
		protected.PATCH("/projects/:id/tasks/:task_id", taskCtl.Update)
		protected.DELETE("/tasks/:id", taskCtl.Delete)
		protected.GET("/projects/:id/tasks/:task_id", taskCtl.Search)
		protected.GET("/tasks", taskCtl.List)
		
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	taskSvc.StartDueWatcher(ctx, logger)
	return r
}
