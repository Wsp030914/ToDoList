package service

import (
	"ToDoList/server/async"
	"ToDoList/server/infra"
	"ToDoList/server/models"
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type AuthService struct {
	bus *async.EventBus
}

func NewAuthService(bus *async.EventBus) *AuthService {
	return &AuthService{bus: bus}
}

func (a *AuthService) ValidateJti(ctx context.Context, lg *zap.Logger, jti string) error {
	ctxRedis, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	blacklisted, err := ExistsJti(ctxRedis, jti)
	if err != nil {
		lg.Warn("user.auth.ExistsJti_redis_failed", zap.Error(err))
		return &AppError{Code: 5001, Message: "服务忙，请稍后重试"}
	}
	if !blacklisted {
		return nil
	}
	return &AppError{Code: 4001, Message: "用户已登出，请重新登录"}
}

func (a *AuthService) ValidateVersion(ctx context.Context, lg *zap.Logger, uid int, reqVersion int) error {
	rctx, rcancel := context.WithTimeout(ctx, 300*time.Millisecond)
	version, cacheErr := GetVersion(rctx, uid)
	rcancel()
	if cacheErr == nil {
		if reqVersion == version {
			return nil
		}
	}

	u, dbErr := models.GetVersionByID(uid)
	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return &AppError{Code: 4001, Message: "该用户不存在"}
		}
		return &AppError{Code: 5001, Message: "服务忙"}
	}
	if reqVersion != u.TokenVersion {
		return &AppError{Code: 4001, Message: "令牌已失效"}
	}

	if errors.Is(cacheErr, redis.Nil) {
		if a.bus != nil {
			infra.Publish(a.bus, lg, "PutVersion", struct {
				UID          int `json:"uid"`
				TokenVersion int `json:"tokenVersion"`
			}{UID: u.ID, TokenVersion: u.TokenVersion}, 100*time.Millisecond, zap.Int("uid", u.ID),
				zap.Int("TokenVersion", u.TokenVersion))
		}
	}
	return nil

}
