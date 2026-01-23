package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)
const (
    ErrCodeOK              = 0     
    ErrCodeAuthFailed      = 4001  
    ErrCodeValidation      = 4002  
    ErrCodeNotFound        = 4004  
    ErrCodeConflict        = 4009  
    ErrCodeInternalServer  = 5001  
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
