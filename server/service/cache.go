package service

import "github.com/redis/go-redis/v9"

type Cache struct {
	Rdb *redis.Client
}

var c Cache

func NewCache(rdb *redis.Client) {
	c.Rdb = rdb
}
