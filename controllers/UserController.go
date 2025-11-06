package controllers

import (
	"NewStudent/auth"
	"NewStudent/dao"
	"NewStudent/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

type UserController struct{}

func (U UserController) Login(c *gin.Context) {
	username := c.DefaultPostForm("username", "")
	password := c.DefaultPostForm("password", "")
	if username == "" || password == "" {
		ReturnError(c, 4001, "请输入正确信息")
		return
	}
	user, err := models.GetUserInfoByUsername(username)
	if err != nil {
		ReturnError(c, 4001, "用户名或密码有误，请重新输入")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		ReturnError(c, 4001, "用户名或密码有误，请重新输入")
		return
	}

	// ✅ 签发 JWT
	token, exp, er := auth.GenerateAccessToken(user.ID, user.Username, user.TokenVersion)
	if er != nil {
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
	username := c.DefaultPostForm("username", "")
	password := c.DefaultPostForm("password", "")
	confirmPassword := c.DefaultPostForm("confirmPassword", "")

	if username == "" || password == "" || confirmPassword == "" {
		ReturnError(c, 4001, "请输入正确信息")
		return
	}
	if password != confirmPassword {
		ReturnError(c, 4001, "密码与确认密码不一致")
		return
	}
	user, err := models.GetUserInfoByUsername(username)
	if user.ID != 0 {
		ReturnError(c, 4001, "用户名已存在")
		return
	}
	_, err = models.AddUser(username, password)
	if err != nil {
		ReturnError(c, 4001, "保存失败，请联系管理员")
		return
	}
	ReturnSuccess(c, 0, "注册成功", username, 1)

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
	res := dao.Db.Model(&models.User{}).
		Where("id = ?", uid).
		UpdateColumn("token_version", gorm.Expr("token_version + 1"))
	if res.Error != nil {
		ReturnError(c, 5000, "登出失败，请稍后重试")
		return
	}
	if res.RowsAffected == 0 {
		ReturnError(c, 4001, "该用户不存在")
		return
	}
	ReturnSuccess(c, 0, "已退出登录", nil, 1)
}
