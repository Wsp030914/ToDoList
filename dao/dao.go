package dao

import (
	"NewStudent/config"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	_ "gorm.io/gorm/logger"
	"time"
)

var (
	Db  *gorm.DB
	err error
)

func init() {
	Db, err = gorm.Open(mysql.Open(config.Dsn), &gorm.Config{})
	if err != nil {
		fmt.Print("mysql connect error", err.Error())
	}
	if Db.Error != nil {
		fmt.Print("database connect error", Db.Error)
	}

	sqlDB, _ := Db.DB()

	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)

}
