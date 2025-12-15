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
)

func main() {
	// 1) 初始化外部依赖
	utils.InitCos()
	if err := initialize.InitMySQL(); err != nil {
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
	r := NewRouter(app)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	dispatcher.Stop()
	_ = srv.Shutdown(context.Background())
}
