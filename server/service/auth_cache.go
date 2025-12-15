package service

import (
	"context"
	"time"
)

func ExistsJti(ctx context.Context, jti string) (bool, error) {
	key := "bl:" + jti
	n, err := c.Rdb.Exists(ctx, key).Result()
	return n == 1, err
}

func PutJti(ctx context.Context, jti string, expire time.Time) error {
	key := "bl:" + jti
	ttl := time.Until(expire)
	if ttl > 0 {
		return c.Rdb.Set(ctx, key, 1, ttl).Err()
	}
	return ErrTokenExpire
}
