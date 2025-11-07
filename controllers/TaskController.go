package controllers

import (
	"NewStudent/models"
	"errors"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

type TaskController struct{}
type CreateTaskRequest struct {
	Title     string     `json:"title" binding:"required,max=200"`
	ProjectID *int       `json:"project_id,omitempty"`
	ContentMD *string    `json:"content_md,omitempty"`
	Priority  *int       `json:"priority,omitempty"`
	Status    *string    `json:"status,omitempty"`
	StartAt   *time.Time `json:"start_at,omitempty"`
	DueAt     *time.Time `json:"due_at,omitempty"`
}

func (T TaskController) Create(c *gin.Context) {
	uid := c.GetInt("uid")

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ReturnError(c, 4001, "请求参数错误")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		ReturnError(c, 4001, "请输入任务名称")
		return
	}
	if req.Priority != nil && (*req.Priority < 1 || *req.Priority > 5) {
		ReturnError(c, 4001, "优先级范围应为 1~5")
		return
	}
	if req.StartAt != nil && req.DueAt != nil && req.DueAt.Before(*req.StartAt) {
		ReturnError(c, 4001, "截止时间不能早于开始时间")
		return
	}

	if req.ProjectID != nil {
		p, err := models.GetProjectByID(uid, req.ProjectID)
		if err != nil {
			ReturnError(c, 4001, err.Error())
		}
		if p.ID == 0 {
			ReturnError(c, 4001, "项目不存在")
			return
		}
	}

	exists, err := models.GetTaskByUserProjectTitle(uid, req.ProjectID, req.Title)
	if err != nil {
		ReturnError(c, 4001, err.Error())
	}
	if exists.ID != 0 {
		ReturnError(c, 4001, "该任务已存在")
		return
	}

	status := "todo"
	if req.Status != nil {
		s := strings.TrimSpace(*req.Status)
		if s != "todo" && s != "doing" && s != "done" && s != "archived" {
			ReturnError(c, 4001, "任务状态错误")
			return
		}
		status = s
	}
	priority := 3
	if req.Priority != nil {
		priority = *req.Priority
	}

	task := models.Task{
		UserID:    uid,
		ProjectID: req.ProjectID,
		Title:     req.Title,
		ContentMD: getString(req.ContentMD),
		Status:    status,
		Priority:  priority,
		StartAt:   req.StartAt,
		DueAt:     req.DueAt,
	}

	created, err := models.CreateTaskByUidAndTask(uid, task)
	if err != nil {
		if errors.Is(err, models.ErrTaskExists) {
			ReturnError(c, 4001, "该任务已存在")
			return
		}
		ReturnError(c, 5000, "创建失败，请稍后重试")
		return
	}
	ReturnSuccess(c, 0, "创建成功", gin.H{
		"task": created,
	}, 1)

}

func getString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
