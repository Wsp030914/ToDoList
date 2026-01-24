package service

import (
	"ToDoList/server/async"
	"ToDoList/server/infra"
	"ToDoList/server/models"
	"ToDoList/server/utils"
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string { return e.Message }

type UserService struct {
	bus *async.EventBus
}

func NewUserService(bus *async.EventBus) *UserService {
	return &UserService{bus: bus}
}

type LoginResult struct {
	AccessToken    string
	AccessExpireAt time.Time
}

func (s *UserService) Login(ctx context.Context, lg *zap.Logger, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	lg = lg.With(zap.String("username", username))
	lg.Info("login.begin")
	
	user, err := models.GetUserInfoByUsername(ctx, username)
	if err != nil {
		lg.Error("login.query_user_failed", zap.Error(err))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "系统错误，请稍后重试"} 
	}
	if user.ID == 0 {
		lg.Warn("login.user_not_found")
		return nil, &AppError{Code: utils.ErrCodeNotFound, Message: "用户不存在"} 
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		lg.Warn("login.password_mismatch")
		return nil, &AppError{Code: utils.ErrCodeAuthFailed, Message: "用户名或密码有误，请重新输入"} 
	}

	token, exp, err := utils.GenerateAccessToken(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		lg.Error("login.jwt_issue_failed", zap.Error(err))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "令牌生成失败"} 
	}
	lg.Info("login.success", zap.Int("uid", user.ID), zap.Time("access_exp", exp))
	return &LoginResult{
		AccessToken:    token,
		AccessExpireAt: exp,
	}, nil
}

type RegisterResult struct {
	User models.User
}

func (s *UserService) Register(ctx context.Context, lg *zap.Logger, email, username, password string, avatarFile *multipart.FileHeader) (*RegisterResult, error) {
	username = strings.TrimSpace(username)
	email = strings.ToLower(strings.TrimSpace(email))
	lg = lg.With(zap.String("username", username), zap.String("email", email))
	lg.Info("register.begin")
	
	exists, err := models.GetUserInfoByUsername(ctx, username)
	if err != nil {
		lg.Error("register.check_username_failed", zap.Error(err))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "系统错误，请稍后再试"} 
	}
	if exists.ID != 0 {
		lg.Info("register.username_exists")
		return nil, &AppError{Code: utils.ErrCodeConflict, Message: "用户名已存在"} 
	}

	exists, err = models.GetUserInfoByEmail(ctx, email)
	if err != nil {
		lg.Error("register.check_email_failed", zap.Error(err))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "系统错误，请稍后再试"} 
	}
	if exists.ID != 0 {
		lg.Info("register.email_exists")
		return nil, &AppError{Code: utils.ErrCodeConflict, Message: "邮箱已被注册"} 
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		lg.Error("register.password_hash_failed", zap.Error(err))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "密码处理失败"} 
	}

	if avatarFile == nil {
		lg.Error("register.avatar_missing")
		return nil, &AppError{Code: utils.ErrCodeValidation, Message: "请上传头像"} 
	}

	avatarKey, _, err := utils.PutObj(ctx, avatarFile)
	if err != nil {
		lg.Error("register.Avatar_post_failed", zap.Error(err))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "头像存储失败"} 
	}

	u := models.User{
		Email:     email,
		Password:  string(hash),
		Username:  username,
		AvatarURL: avatarKey,
	}

	created, err := models.AddUser(ctx, u)
	if err != nil {
		if errors.Is(err, models.ErrUserExists) {
			lg.Info("register.duplicate_on_insert")
			return nil, &AppError{Code: utils.ErrCodeConflict, Message: "该用户已存在"}
		}
		lg.Error("register.insert_failed", zap.Error(err))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "保存失败，请联系管理员"}
	}
	infra.Publish(s.bus, lg, "PutAvatar", struct {
		UID       int    `json:"uid"`
		AvatarKey string `json:"avatarKey"`
	}{UID: created.ID, AvatarKey: avatarKey}, 300*time.Millisecond, zap.Int("uid", created.ID))
	lg.Info("register.success", zap.Int("uid", created.ID))

	return &RegisterResult{User: created}, nil
}

// Logout 登出
func (s *UserService) Logout(ctx context.Context, lg *zap.Logger, uid int, claims *utils.Claims) error {
	if uid <= 0 || claims == nil {
		lg.Warn("logout.invalid_input", zap.Int("uid", uid))
		return &AppError{Code: utils.ErrCodeAuthFailed, Message: "未授权"}
	}
	lg = lg.With(zap.Int("uid", uid))
	lg.Info("logout.begin")
	
	ctxRedis, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	err := PutJti(ctxRedis, claims.RegisteredClaims.ID, claims.RegisteredClaims.ExpiresAt.Time)
	if err != nil {
		if errors.Is(err, ErrTokenExpire) {
			lg.Warn("logout.ErrTokenExpire")
			return &AppError{Code: utils.ErrCodeAuthFailed, Message: "token到期"}
		}
		lg.Warn("logout.redis_put_error", zap.Error(err))
		return &AppError{Code:utils.ErrCodeInternalServer, Message: "写入Redis出错"}
	}
	lg.Info("logout.success")
	return nil
}

type UpdateUserInput struct {
	Email           *string
	Username        *string
	Password        *string
	ConfirmPassword *string
	AvatarFile      *multipart.FileHeader
}
type TokenInfo struct {
	AccessToken    string
	AccessExpireAt time.Time
}
type UpdateUserResult struct {
	User     models.User
	Affected int64
	Token    *TokenInfo
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, lg *zap.Logger, uid int, in UpdateUserInput) (*UpdateUserResult, error) {
	lg = lg.With(zap.Int("uid", uid))
	lg.Info("update.user.begin")
	
	update := map[string]interface{}{}
	if in.Username != nil && strings.TrimSpace(*in.Username) != "" {
		username := strings.TrimSpace(*in.Username)
		exists, err := models.GetUserInfoByUsername(ctx, username)
		if err != nil {
			lg.Error("user.update.check_username_failed", zap.Error(err))
			return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "请稍后再试：" + err.Error()}
		}
		if exists.ID != uid && exists.ID != 0 {
			lg.Info("user.update.username_exists", zap.String("username", username))
			return nil, &AppError{Code: utils.ErrCodeConflict, Message: "用户已存在"}
		}
		update["username"] = username
	}
	if in.Email != nil && strings.TrimSpace(*in.Email) != "" {
		email := strings.ToLower(strings.TrimSpace(*in.Email))
		exists, err := models.GetUserInfoByEmail(ctx, email)
		if err != nil {
			lg.Error("user.update.check_email_failed", zap.Error(err))
			return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "请稍后再试：" + err.Error()}
		}
		if exists.ID != uid && exists.ID != 0 {
			lg.Info("user.update.email_exists", zap.String("email", email))
			return nil, &AppError{Code: utils.ErrCodeConflict, Message: "邮箱已存在"}
		}
		update["email"] = email
	}

	oldKey := ""
	newKey := ""
	if in.AvatarFile != nil {
		oldUser, err := models.GetUserInfoByID(ctx, uid)
		if err != nil {
			lg.Error("user.update.get_old_avatar_failed", zap.Error(err))
			return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "请稍后再试：" + err.Error()}
		}
		oldKey = strings.TrimSpace(oldUser.AvatarURL)

		key, _, err := utils.PutObj(ctx, in.AvatarFile)
		if err != nil {
			lg.Warn("user.update.avatar_put_failed", zap.Error(err))
			return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "更新头像失败，请重新再试"}
		}
		newKey = key
		update["avatar_url"] = newKey
	}

	if in.Password != nil && in.ConfirmPassword != nil {
		if *in.Password != *in.ConfirmPassword {
			lg.Warn("user.update.password_mismatch")
			return nil, &AppError{Code: utils.ErrCodeValidation, Message: "两次输入密码不一致，请重新输入"}
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*in.Password), bcrypt.DefaultCost)
		if err != nil {
			lg.Error("user.update.password_hash_failed", zap.Error(err))
			return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "密码处理失败"}
		}
		update["password"] = string(hash)
		update["token_version"] = gorm.Expr("token_version + 1")
	} else if in.Password != nil || in.ConfirmPassword != nil {
		lg.Warn("user.update.password_half_provided")
		return nil, &AppError{Code: utils.ErrCodeValidation, Message: "请同时提供密码与确认密码"}
	}

	if len(update) == 0 {
		lg.Info("user.update.no_fields")
		return nil, &AppError{Code: utils.ErrCodeValidation, Message: "没有需要更新的字段"}
	}

	updated, err, affected := models.UpdateUser(ctx, update, uid)
	if err != nil {
		if newKey != "" && s.bus != nil {
			infra.Publish(s.bus, lg, "DeleteCOS", struct {
				Key string `json:"key"`
			}{Key: newKey}, 300*time.Millisecond, zap.Int("uid", uid),
				zap.String("COSKey", newKey))

		}
		if errors.Is(err, models.ErrUserExists) {
			lg.Error("user.update.duplicate_on_update", zap.Any("update", sanitize(update)))
			return nil, &AppError{Code: utils.ErrCodeConflict, Message: "用户已存在"}
		}
		lg.Error("user.update.db_failed", zap.Error(err), zap.Any("update", sanitize(update)))
		return nil, &AppError{Code: utils.ErrCodeInternalServer, Message: "更新失败，请稍后重试"}
	}
	if newKey != "" && oldKey != "" && oldKey != newKey {
		if s.bus != nil {
			infra.Publish(s.bus, lg, "DeleteCOS", struct {
				Key string `json:"key"`
			}{Key: oldKey}, 300*time.Millisecond, zap.Int("uid", uid),
				zap.String("COSKey", oldKey))
		}
	}
	if newKey != "" {
		if s.bus != nil {
			infra.Publish(s.bus, lg, "UpdateAvatar", struct {
				UID       int    `json:"uid"`
				AvatarKey string `json:"avatarKey"`
			}{UID: updated.ID, AvatarKey: newKey}, 100*time.Millisecond, zap.Int("uid", updated.ID),
				zap.String("AvatarKey", newKey))
		}
	}
	if affected == 0 {
		lg.Info("user.update.noop")
		return &UpdateUserResult{
			User:     updated,
			Affected: affected,
			Token:    nil,
		}, nil
	}
	// 不需要刷新 token
	if _, ok := update["token_version"]; !ok {
		lg.Info("user.update.success", zap.Int64("affected", affected))
		return &UpdateUserResult{
			User:     updated,
			Affected: affected,
			Token:    nil,
		}, nil
	}

	// 更新 redis token_version 并签发新 token
	ctxRedis, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	if err := PutVersion(ctxRedis, updated.ID, updated.TokenVersion); err != nil {
		lg.Warn("user.update.putTokenVersion_redis_failed", zap.Error(err))
	}

	tokenStr, exp, _ := utils.GenerateAccessToken(updated.ID, updated.Username, updated.TokenVersion)
	lg.Info("user.update.password_changed", zap.Time("new_access_exp", exp))
	return &UpdateUserResult{
		User:     updated,
		Affected: affected,
		Token: &TokenInfo{
			AccessToken:    tokenStr,
			AccessExpireAt: exp,
		},
	}, nil
}

func sanitize(m map[string]interface{}) map[string]interface{} {
	cp := map[string]interface{}{}
	for k, v := range m {
		if k == "password" {
			cp[k] = "***redacted***"
		} else {
			cp[k] = v
		}
	}
	return cp
}
