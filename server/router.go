package main

import (
	"ToDoList/server/async"
	"ToDoList/server/config"
	"ToDoList/server/handler"
	"ToDoList/server/middlewares"
	"ToDoList/server/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

type App struct {
	Bus *async.EventBus
	Rdb *redis.Client
	Db  *gorm.DB
}

func NewRouter(app *App) *gin.Engine {
	cfg, err := config.LoadZapConfig()
	if err != nil {
		panic(err)
	}
	logger := config.InitZap(cfg)

	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	r := gin.New()
	r.Use(middlewares.AccessLogMiddleware(), middlewares.RecoveryWithZap())

	userSvc := service.NewUserService(app.Bus)
	userCtl := handler.NewUserHandler(userSvc)
	projectSvc := service.NewProjectService(app.Bus)
	projectCtl := handler.NewProjectHandler(projectSvc)
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
	}

	return r
}
