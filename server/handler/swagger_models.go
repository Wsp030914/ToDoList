package handler

import (
	"ToDoList/server/models"
	"ToDoList/server/service"
)

type ErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type LoginData struct {
	AccessToken     string `json:"access_token"`
	TokenType       string `json:"token_type"`
	AccessExpiresAt string `json:"access_expires_at"`
}

type LoginResponse struct {
	Code  int       `json:"code"`
	Msg   string    `json:"msg"`
	Data  LoginData `json:"data"`
	Count int64     `json:"count"`
}

type RegisterResponse struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Data  models.User `json:"data"`
	Count int64       `json:"count"`
}

type LogoutResponse struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
	Count int64       `json:"count"`
}

type UpdateUserResponse struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
	Count int64       `json:"count"`
}

type ProjectDetailData struct {
	Project service.ProjectProfile `json:"project"`
}

type ProjectDetailResponse struct {
	Code  int               `json:"code"`
	Msg   string            `json:"msg"`
	Data  ProjectDetailData `json:"data"`
	Count int64             `json:"count"`
}

type ProjectListData struct {
	List     []service.ProjectSummary `json:"list"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
	Total    int64                    `json:"total"`
}

type ProjectListResponse struct {
	Code  int             `json:"code"`
	Msg   string          `json:"msg"`
	Data  ProjectListData `json:"data"`
	Count int64           `json:"count"`
}

type ProjectCreateData struct {
	Project models.Project `json:"project"`
}

type ProjectCreateResponse struct {
	Code  int               `json:"code"`
	Msg   string            `json:"msg"`
	Data  ProjectCreateData `json:"data"`
	Count int64             `json:"count"`
}

type ProjectUpdateData struct {
	Project models.Project `json:"project"`
}

type ProjectUpdateResponse struct {
	Code  int               `json:"code"`
	Msg   string            `json:"msg"`
	Data  ProjectUpdateData `json:"data"`
	Count int64             `json:"count"`
}

type ProjectDeleteData struct {
	ID           int   `json:"id"`
	TaskAffected int64 `json:"task_affected"`
	ProjAffected int64 `json:"proj_affected"`
}

type ProjectDeleteResponse struct {
	Code  int               `json:"code"`
	Msg   string            `json:"msg"`
	Data  ProjectDeleteData `json:"data"`
	Count int64             `json:"count"`
}

type TaskCreateData struct {
	Task models.Task `json:"task"`
}

type TaskCreateResponse struct {
	Code  int            `json:"code"`
	Msg   string         `json:"msg"`
	Data  TaskCreateData `json:"data"`
	Count int64          `json:"count"`
}

type TaskUpdateResponse struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Data  models.Task `json:"data"`
	Count int64       `json:"count"`
}

type TaskDeleteData struct {
	ID           int   `json:"id"`
	TaskAffected int64 `json:"task_affected"`
}

type TaskDeleteResponse struct {
	Code  int            `json:"code"`
	Msg   string         `json:"msg"`
	Data  TaskDeleteData `json:"data"`
	Count int64          `json:"count"`
}

type TaskDetailResponse struct {
	Code  int                `json:"code"`
	Msg   string             `json:"msg"`
	Data  service.TaskDetail `json:"data"`
	Count int64              `json:"count"`
}

type TaskListData struct {
	List     []service.TaskSummary `json:"list"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Total    int64                 `json:"total"`
}

type TaskListResponse struct {
	Code  int          `json:"code"`
	Msg   string       `json:"msg"`
	Data  TaskListData `json:"data"`
	Count int64        `json:"count"`
}
