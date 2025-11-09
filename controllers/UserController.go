package controllers

import (
	"NewStudent/auth"
	"NewStudent/models"
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"strings"
	"time"
)

type UserController struct{}
type RegisterReq struct {
	Email           string  `json:"email"            binding:"required,email,max=255"`
	Username        string  `json:"username"         binding:"required,min=2,max=64"`
	Password        string  `json:"password"         binding:"required,min=8,max=72"`
	ConfirmPassword string  `json:"confirm_password"  binding:"required,eqfield=Password"`
	AvatarURL       *string `json:"avatar_url,omitempty" binding:"omitempty,url,max=512"`
}
type LoginReq struct {
	Username string `json:"username"         binding:"required,min=2,max=64"`
	Password string `json:"password"         binding:"required,min=8,max=72"`
}

type UpdateUserReq struct {
	Email           *string `json:"email,omitempty"    binding:"omitempty,email,max=255"`
	Username        *string `json:"username,omitempty" binding:"omitempty,min=2,max=64"`
	Password        *string `json:"password,omitempty"        binding:"omitempty,min=8,max=72"`
	ConfirmPassword *string `json:"confirm_password,omitempty" binding:"omitempty,eqfield=Password"`
	//OldPassword     *string `json:"old_password,omitempty" binding:"omitempty,eqfield=Password"`
	AvatarURL *string `json:"avatar_url,omitempty" binding:"omitempty,url,max=512"`
}

func (U UserController) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ReturnError(c, 4001, "参数格式有误，请重新输入"+err.Error())
		return
	}
	username := strings.TrimSpace(req.Username)
	user, err := models.GetUserInfoByUsername(username)
	if err != nil {
		ReturnError(c, 4001, "请稍后重试: "+err.Error())
		return
	}
	if user.ID == 0 || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		ReturnError(c, 4001, "用户名或密码有误，请重新输入")
		return
	}

	// ✅ 签发 JWT
	token, exp, err := auth.GenerateAccessToken(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		ReturnError(c, 4001, "签发令牌失败")
		return
	}
	ReturnSuccess(c, 0, "登陆成功", gin.H{
		"access_token":      token,
		"token_type":        "Bearer",
		"access_expires_at": exp.UTC().Format(time.RFC3339),
	}, 1)
}

func (U UserController) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ReturnError(c, 4001, "参数格式错误："+err.Error())
		return
	}
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	exists, err := models.GetUserInfoByUsername(username)
	if err != nil {
		ReturnError(c, 4001, "请稍后再试："+err.Error())
		return
	}
	if exists.ID != 0 {
		ReturnError(c, 4001, "用户已存在")
		return
	}
	exists, err = models.GetUserInfoByEmail(email)
	if err != nil {
		ReturnError(c, 4001, "请稍后再试："+err.Error())
		return
	}
	if exists.ID != 0 {
		ReturnError(c, 4001, "邮箱已存在")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ReturnError(c, 4001, "密码处理失败")
		return
	}
	password := string(hash)

	user := models.User{
		Email:    email,
		Password: password,
		Username: username,
	}

	if req.AvatarURL != nil && strings.TrimSpace(*req.AvatarURL) != "" {
		user.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}

	created, err := models.AddUser(user)
	if err != nil {
		if errors.Is(err, models.ErrUserExists) {
			ReturnError(c, 4001, "该用户已存在")
			return
		}
		ReturnError(c, 4001, "保存失败，请联系管理员")
		return
	}
	ReturnSuccess(c, 0, "注册成功", created, 1)

}

func (U UserController) Logout(c *gin.Context) {
	uidAny, ok := c.Get("uid")
	if !ok {
		ReturnError(c, 4001, "未授权")
		return
	}
	uid, ok := uidAny.(int)
	if !ok || uid == 0 {
		ReturnError(c, 4001, "未授权")
		return
	}
	err, affected := models.LogoutUser(uid)
	if err != nil {
		ReturnError(c, 5000, "登出失败，请稍后重试")
		return
	}
	if affected == 0 {
		ReturnError(c, 4001, "该用户不存在")
		return
	}
	ReturnSuccess(c, 0, "已退出登录", nil, 1)
}

func (U UserController) Update(c *gin.Context) {
	uid := c.GetInt("uid")

	update := map[string]interface{}{}
	var req UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		ReturnError(c, 4001, "参数格式有误，请重新输入"+err.Error())
		return
	}

	if req.Username != nil && strings.TrimSpace(*req.Username) != "" {
		username := strings.TrimSpace(*req.Username)
		exists, err := models.GetUserInfoByUsername(username)
		if err != nil {
			ReturnError(c, 4001, "请稍后再试："+err.Error())
			return
		}
		if exists.ID != uid && exists.ID != 0 {
			ReturnError(c, 4001, "用户已存在")
			return
		}
		update["username"] = username

	}

	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		exists, err := models.GetUserInfoByEmail(email)
		if err != nil {
			ReturnError(c, 4001, "请稍后再试："+err.Error())
			return
		}
		if exists.ID != uid && exists.ID != 0 {
			ReturnError(c, 4001, "邮箱已存在")
			return
		}
		update["email"] = email
	}

	if req.AvatarURL != nil {
		avatarURL := strings.TrimSpace(*req.AvatarURL)
		update["avatar_url"] = avatarURL
	}

	if req.Password != nil && req.ConfirmPassword != nil {
		if *req.Password != *req.ConfirmPassword {
			ReturnError(c, 4001, "两次输入密码不一致，请重新输入")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			ReturnError(c, 4001, "密码处理失败")
			return
		}
		password := string(hash)
		update["password"] = password
		update["token_version"] = gorm.Expr("token_version + 1")
	} else if req.Password != nil || req.ConfirmPassword != nil {
		ReturnError(c, 4001, "请同时提供密码与确认密码")
		return
	}

	if len(update) == 0 {
		ReturnError(c, 4001, "没有需要更新的字段")
		return
	}

	updated, err, affected := models.UpdateUser(update, uid)
	if err != nil {
		if errors.Is(err, models.ErrUserExists) {
			ReturnError(c, 4001, "用户已存在")
			return
		}
		ReturnError(c, 4001, "更新失败，请稍后重试")
		return
	}
	if affected == 0 {
		ReturnSuccess(c, 0, "未修改任何字段", updated, affected)
		return
	}
	if _, ok := update["token_version"]; !ok {
		ReturnSuccess(c, 0, "更新成功", updated, affected)
		return
	}
	token, exp, _ := auth.GenerateAccessToken(updated.ID, updated.Username, updated.TokenVersion)
	ReturnSuccess(c, 0, "密码已更新，已退出其他设备", gin.H{
		"access_token":      token,
		"token_type":        "Bearer",
		"access_expires_at": exp.UTC().Format(time.RFC3339),
		"user":              updated, // 按需
	}, affected)

}
