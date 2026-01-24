package main

import (
	"ToDoList/server/async"
	"ToDoList/server/initialize"
	"ToDoList/server/models"
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	_ "ToDoList/server/docs"
)

// @title ToDoList API
// @version 1.0
// @description 管理API
// @basePath /api/v1
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()
	utils.InitCos()
	if err := initialize.InitMySQL(); err != nil {
		panic(err)
	}
	if err := initialize.Db.AutoMigrate(&models.User{}, &models.Task{}, &models.Project{}); err != nil {
		panic(err)
	}

	if err := initialize.InitRedis(); err != nil {
		panic(err)
	}
	models.NewDB(initialize.Db)
	service.NewCache(initialize.Rdb)
	dispatcher := async.NewDispatcher(256)
	dispatcher.Start(4)
	bus := async.NewEventBus(dispatcher)

	initialize.InitAsyncHandlers(dispatcher)
	app := &App{Bus: bus, Rdb: initialize.Rdb, Db: initialize.Db}
	r := NewRouter(ctx, app)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	
	<-ctx.Done()
	dispatcher.Stop()

	_ = srv.Shutdown(context.Background())
}
