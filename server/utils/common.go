package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)
const (
    CodeOK              = 0     
    ErrCodeAuthFailed      = 4001  //认证失败（无效的JWT/未登录）
    ErrCodeValidation      = 4002  //参数验证失败（email格式错、密码过短等）
    ErrCodeNotFound        = 4004  //资源未找到（用户不存在、项目不存在）
    ErrCodeConflict        = 4009  //冲突（邮箱已被注册、用户名已存在）
    ErrCodeInternalServer  = 5001  //服务器内部错误（数据库异常、系统错误）
)
type JsonStruct struct {
	Code  int         `json:"code"`
	Msg   interface{} `json:"msg"`
	Data  interface{} `json:"data"`
	Count int64       `json:"count"`
}
type JsonErrStruct struct {
	Code int         `json:"code"`
	Msg  interface{} `json:"msg"`
}

func ReturnSuccess(c *gin.Context, code int, msg interface{}, data interface{}, count int64) {
	statusCode := http.StatusOK
	json := &JsonStruct{
		Code:  code,
		Msg:   msg,
		Data:  data,
		Count: count,
	}
	c.JSON(statusCode, json)
}

func ReturnError(c *gin.Context, code int, msg interface{}) {
    var statusCode int
    switch code {
    case ErrCodeAuthFailed:
        statusCode = http.StatusUnauthorized  
    case ErrCodeValidation:
        statusCode = http.StatusBadRequest   
    case ErrCodeNotFound:
        statusCode = http.StatusNotFound      
    case ErrCodeConflict:
        statusCode = http.StatusConflict      
    case ErrCodeInternalServer:
        statusCode = http.StatusInternalServerError  
    default:
        statusCode = http.StatusOK  
    }

    json := &JsonErrStruct{
        Code: code,
        Msg:  msg,
    }
    c.JSON(statusCode, json)
}
