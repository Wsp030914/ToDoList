package initialize

import (
	"ToDoList/server/config"
	"context"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	_ "gorm.io/gorm/logger"
	"time"
)

var Db *gorm.DB

func InitMySQL() error {
	dsn, err := config.LoadMysqlConfig()
	if err != nil {
		return err
	}
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Print("mysql connect error", err.Error())
	}
	sqlDB, err := Db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("mysql ping failed: %w", err)
	}

	return nil
}
