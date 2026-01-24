package handler

import (
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"errors"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
)

type RegisterReq struct {
	Email           string `json:"email" form:"email" binding:"required,email,max=255"`
	Username        string `json:"username" form:"username" binding:"required,min=2,max=64"`
	Password        string `json:"password" form:"password" binding:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password" binding:"required,eqfield=Password"`
}

type LoginReq struct {
	Username string `json:"username" binding:"required,min=2,max=64"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type UpdateUserReq struct {
	Email           *string `json:"email" form:"email" binding:"omitempty,email,max=255"`
	Username        *string `json:"username" form:"username" binding:"omitempty,min=2,max=64"`
	Password        *string `json:"password" form:"password" binding:"omitempty,min=8,max=72,required_with=ConfirmPassword"`
	ConfirmPassword *string `json:"confirm_password" form:"confirm_password" binding:"omitempty,required_with=Password,eqfield=Password"`
}

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// @Summary 用户登录
// @Description 使用用户名和密码进行身份验证，获取JWT token
// @Accept json
// @Produce json
// @Param body body LoginReq true "登录请求参数"
// @Success 200 {object} LoginResponse "登录成功，返回token"
// @Failure 400 {object} ErrorResponse "参数错误"
// @Failure 401 {object} ErrorResponse "登录失败"
// @Failure 404 {object} ErrorResponse "用户不存在"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /login [post]
func (u *UserHandler) Login(c *gin.Context) {
	lg := utils.CtxLogger(c)
	start := time.Now()
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("user.login.param_bind_failed", zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "参数格式有误，请重新输入")
		return
	}
	lg = lg.With(zap.String("username", req.Username))

	res, err := u.svc.Login(c.Request.Context(), lg, req.Username, req.Password)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			lg.Warn("user.login.failed", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Error("user.login.error", zap.Error(err), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}

	lg.Info("user.login.success", zap.Duration("elapsed_ms", time.Since(start)))
	utils.ReturnSuccess(c, utils.CodeOK, "登陆成功", gin.H{
		"access_token":      res.AccessToken,
		"token_type":        "Bearer",
		"access_expires_at": res.AccessExpireAt.UTC().Format(time.RFC3339),
	}, 1)
}

// @Summary 用户注册
// @Description 使用邮箱、用户名、密码和头像进行注册
// @Accept multipart/form-data
// @Produce json
// @Param email formData string true "邮箱地址"
// @Param username formData string true "用户名（2-64字符）"
// @Param password formData string true "密码（8-72字符）"
// @Param confirm_password formData string true "确认密码"
// @Param file formData file true "头像文件"
// @Success 200 {object} RegisterResponse "注册成功，返回用户信息"
// @Failure 400 {object} ErrorResponse "参数错误或头像上传失败"
// @Failure 409 {object} ErrorResponse "用户已存在"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /register [post]
func (u *UserHandler) Register(c *gin.Context) {
	lg := utils.CtxLogger(c)
	start := time.Now()

	var req RegisterReq
	if err := c.ShouldBindWith(&req, binding.FormMultipart); err != nil {
		lg.Warn("user.register.param_bind_failed", zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "参数格式错误："+err.Error())
		return
	}
	lg = lg.With(zap.String("username", req.Username), zap.String("email", req.Email))
	fh, err := c.FormFile("file")
	if err != nil {
		lg.Warn("user.register.avatar_missing", zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "头像上传失败："+err.Error())
		return
	}
	res, err := u.svc.Register(c.Request.Context(), lg, req.Email, req.Username, req.Password, fh)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			lg.Warn("user.register.failed", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Error("user.register.error", zap.Error(err), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}

	lg.Info("user.register.success", zap.Int("uid", res.User.ID), zap.Duration("elapsed_ms", time.Since(start)))
	utils.ReturnSuccess(c, utils.CodeOK, "注册成功", res.User, 1)
}

// @Summary 用户登出
// @Description 需要有效的JWT token认证，无需请求体
// @Produce json
// @Security Bearer
// @Success 200 {object} LogoutResponse "退出登录成功"
// @Failure 401 {object} ErrorResponse "未授权或token无效"
// @Failure 500 {object} ErrorResponse "写入Redis出错"
// @Router /logout [post]
func (u *UserHandler) Logout(c *gin.Context) {
	lg := utils.CtxLogger(c)
	start := time.Now()

	uidAny, ok := c.Get("uid")
	if !ok {
		lg.Warn("user.logout.uid_missing")
		utils.ReturnError(c, utils.ErrCodeAuthFailed, "缺失用户参数")
		return
	}
	uid, ok := uidAny.(int)
	if !ok || uid <= 0 {
		lg.Warn("user.logout.uid_invalid", zap.Any("uid_any", uidAny))
		utils.ReturnError(c, utils.ErrCodeAuthFailed, "用户参数格式出错")
		return
	}
	lg = lg.With(zap.Int("uid", uid))
	v, ok := c.Get("claims")
	if !ok {
		lg.Warn("user.logout.claims_missing")
		utils.ReturnError(c, utils.ErrCodeAuthFailed, "用户未授权")
		return
	}
	claims, ok := v.(*utils.Claims)
	if !ok {
		lg.Warn("user.logout.claims_invalid", zap.Any("claims_type", v))
		utils.ReturnError(c, utils.ErrCodeAuthFailed, "用户未授权")
		return
	}

	if err := u.svc.Logout(c.Request.Context(), lg, uid, claims); err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			lg.Warn("user.logout.failed", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Error("user.logout.error", zap.Error(err), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}

	lg.Info("user.logout.success", zap.Duration("elapsed_ms", time.Since(start)))
	utils.ReturnSuccess(c, utils.CodeOK, "已退出登录", nil, 1)
}

// @Summary 更新用户信息
// @Description 更新用户邮箱、用户名、密码和头像（头像可选）
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param email formData string false "邮箱地址"
// @Param username formData string false "用户名（2-64字符）"
// @Param password formData string false "新密码（8-72字符）"
// @Param confirm_password formData string false "确认新密码"
// @Param file formData file false "头像文件（可选）"
// @Success 200 {object} UpdateUserResponse "更新成功,如更新密码则刷新token"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 400 {object} ErrorResponse "参数错误"
// @Failure 409 {object} ErrorResponse "用户名或邮箱已存在"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /users/me [patch]
func (u *UserHandler) Update(c *gin.Context) {
	lg := utils.CtxLogger(c)
	start := time.Now()
	uid := c.GetInt("uid")
	lg = lg.With(zap.Int("uid", uid))
	var req UpdateUserReq
	if err := c.ShouldBindWith(&req, binding.FormMultipart); err != nil {
		lg.Warn("user.update.param_bind_failed", zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "参数格式有误，请重新输入"+err.Error())
		return
	}

	var fh *multipart.FileHeader
	file, err := c.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			fh = nil
		} else {
			lg.Warn("user.update.avatar_read_failed", zap.Error(err))
			utils.ReturnError(c, utils.ErrCodeValidation, "头像上传失败："+err.Error())
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
			lg.Warn("user.update.failed", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Error("user.update.error", zap.Error(err), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}
	if res.Token == nil {
		lg.Info("user.update.success", zap.Int64("affected", res.Affected), zap.Duration("elapsed_ms", time.Since(start)))
		utils.ReturnSuccess(c, utils.CodeOK, "更新成功", res.User, res.Affected)
		return
	}
	lg.Info("user.update.success_with_token_refresh", zap.Int64("affected", res.Affected), zap.Duration("elapsed_ms", time.Since(start)))
	utils.ReturnSuccess(c, utils.CodeOK, "信息已更新", gin.H{
		"access_token":      res.Token.AccessToken,
		"token_type":        "Bearer",
		"access_expires_at": res.Token.AccessExpireAt.UTC().Format(time.RFC3339),
		"user":              res.User,
	}, res.Affected)
}
