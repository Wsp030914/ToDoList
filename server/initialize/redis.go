package initialize

import (
	"ToDoList/server/config"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

var Rdb *redis.Client

func InitRedis() error {
	redisConfig, err := config.LoadRedisConfig()
	if err != nil {
		return err
	}
	Rdb = redis.NewClient(&redis.Options{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := Rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}
