package controllers

import (
	"NewStudent/auth"
	"NewStudent/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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
	_, _, err = auth.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		ReturnError(c, 5000, "签发令牌失败")
		return
	}
	ReturnSuccess(c, 0, "登陆成功", username, 1)
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
