package service

import (
	"context"
	"errors"
	"strconv"
	"time"
)

var ErrTokenExpire = errors.New("token到期")

// PutAvatar redis缓存头像key
func UpdateAvatarKey(ctx context.Context, userId int, avatarKey string) error {
	sUserid := strconv.Itoa(userId)
	return c.Rdb.Set(ctx, sUserid, avatarKey, 24*time.Hour).Err()
}

//PutVersion redis缓存登录token_version
func PutVersion(ctx context.Context, userID int, tokenVersion int) error {
	key := "uver:" + strconv.Itoa(userID)
	return c.Rdb.Set(ctx, key, tokenVersion, 24*time.Hour).Err()
}

//GetVersion redis取token_version
func GetVersion(ctx context.Context, userId int) (int, error) {
	key := "uver:" + strconv.Itoa(userId)
	val, err := c.Rdb.Do(ctx, "get", key).Int()
	return val, err
}

func PutTraceID(ctx context.Context, jobType string, traceID string, err error) {
	key := "job_done:" + jobType + ":" + traceID
	if err == nil {
		c.Rdb.Set(ctx, key, "ok", 30*time.Minute)
	} else {
		c.Rdb.Set(ctx, key, "fail", 30*time.Minute)
	}
}
