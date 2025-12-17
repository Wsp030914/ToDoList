package handler

import (
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"mime/multipart"
	"net/http"
	"time"
)

type RegisterReq struct {
	Email           string `form:"email"            binding:"required,email,max=255"`
	Username        string `form:"username"         binding:"required,min=2,max=64"`
	Password        string `form:"password"         binding:"required,min=8,max=72"`
	ConfirmPassword string `form:"confirm_password" binding:"required,eqfield=Password"`
}

type LoginReq struct {
	Username string `form:"username" binding:"required,min=2,max=64"`
	Password string `form:"password" binding:"required,min=8,max=72"`
}

type UpdateUserReq struct {
	Email           *string `form:"email"    binding:"omitempty,email,max=255"`
	Username        *string `form:"username" binding:"omitempty,min=2,max=64"`
	Password        *string `form:"password" binding:"omitempty,min=8,max=72,required_with=ConfirmPassword"`
	ConfirmPassword *string `form:"confirm_password" binding:"omitempty,required_with=Password,eqfield=Password"`
}

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (u *UserHandler) Login(c *gin.Context) {
	lg := utils.CtxLogger(c)

	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		lg.Warn("login.bind_failed", zap.Error(err))
		utils.ReturnError(c, 4001, "参数格式有误，请重新输入")
		return
	}

	res, err := u.svc.Login(c.Request.Context(), lg, req.Username, req.Password)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 4001, "系统错误")
		}
		return
	}

	utils.ReturnSuccess(c, 0, "登陆成功", gin.H{
		"access_token":      res.AccessToken,
		"token_type":        "Bearer",
		"access_expires_at": res.AccessExpireAt.UTC().Format(time.RFC3339),
	}, 1)
}

func (u *UserHandler) Register(c *gin.Context) {
	lg := utils.CtxLogger(c)

	var req RegisterReq
	if err := c.ShouldBind(&req); err != nil {
		lg.Warn("register.bind_failed", zap.Error(err))
		utils.ReturnError(c, 4001, "参数格式错误："+err.Error())
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		lg.Error("register.Avatar_post_failed", zap.Error(err))
		utils.ReturnError(c, 4001, "头像上传失败")
		return
	}

	res, err := u.svc.Register(c.Request.Context(), lg, req.Email, req.Username, req.Password, fh)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 4001, "系统错误")
		}
		return
	}

	utils.ReturnSuccess(c, 0, "注册成功", res.User, 1)
}

// Logout 用户登出
func (u *UserHandler) Logout(c *gin.Context) {
	lg := utils.CtxLogger(c)

	uidAny, ok := c.Get("uid")
	if !ok {
		lg.Warn("logout.no_uid_in_ctx")
		utils.ReturnError(c, 4001, "未授权")
		return
	}
	uid, ok := uidAny.(int)
	if !ok || uid == 0 {
		lg.Warn("logout.bad_uid_type", zap.Any("uidAny", uidAny))
		utils.ReturnError(c, 4001, "未授权")
		return
	}

	v, ok := c.Get("claims")
	if !ok {
		lg.Warn("logout.no_claims_in_ctx")
		utils.ReturnError(c, 4001, "未授权")
		return
	}
	claims, ok := v.(*utils.Claims)
	if !ok {
		lg.Warn("logout.bad_claims_type", zap.Any("claims", v))
		utils.ReturnError(c, 4001, "未授权")
		return
	}

	if err := u.svc.Logout(c.Request.Context(), lg, uid, claims); err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 4001, "系统错误")
		}
		return
	}

	utils.ReturnSuccess(c, 0, "已退出登录", nil, 1)
}

// Update 更新用户信息
func (u *UserHandler) Update(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	lg = lg.With(zap.Int("uid", uid))

	var req UpdateUserReq
	if err := c.ShouldBind(&req); err != nil {
		lg.Warn("user.update.bind_failed", zap.Error(err))
		utils.ReturnError(c, 4001, "参数格式有误，请重新输入"+err.Error())
		return
	}

	var fh *multipart.FileHeader
	file, err := c.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			fh = nil
		} else {
			utils.ReturnError(c, 4001, "头像上传失败："+err.Error())
			return
		}
	} else {
		fh = file
	}

	in := service.UpdateUserInput{
		Email:           req.Email,
		Username:        req.Username,
		Password:        req.Password,
		ConfirmPassword: req.ConfirmPassword,
		AvatarFile:      fh,
	}

	res, err := u.svc.UpdateUser(c.Request.Context(), lg, uid, in)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 4001, "系统错误")
		}
		return
	}

	// affected == 0 的情况，在 service 里已经按 success 处理
	if res.Token == nil {
		utils.ReturnSuccess(c, 0, "更新成功", res.User, res.Affected)
		return
	}

	// 修改密码，需要带新 token
	utils.ReturnSuccess(c, 0, "信息已更新", gin.H{
		"access_token":      res.Token.AccessToken,
		"token_type":        "Bearer",
		"access_expires_at": res.Token.AccessExpireAt.UTC().Format(time.RFC3339),
		"user":              res.User,
	}, res.Affected)
}
