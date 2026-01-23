package handler

import (
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TaskHandler struct {
	svc *service.TaskService
}
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}
type CreateTaskRequest struct {
	Title     string     `json:"title" binding:"required,max=200"`
	ProjectID int        `json:"project_id"`
	ContentMD *string    `json:"content_md"`
	Priority  *int       `json:"priority"`
	Status    *string    `json:"status"`
	DueAt     *time.Time `json:"due_at"`
}

func (t *TaskHandler) Create(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("task.create.bind_failed", zap.Error(err))
		utils.ReturnError(c, 4001, "请求参数错误")
		return
	}

	in := service.CreateTaskInput{
		Title:     req.Title,
		ProjectID: req.ProjectID,
		ContentMD: req.ContentMD,
		Priority:  req.Priority,
		Status:    req.Status,
		DueAt:     req.DueAt,
	}

	created, err := t.svc.Create(c.Request.Context(), lg, uid, in)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
		return
	}
	utils.ReturnSuccess(c, 0, "创建成功", gin.H{
		"task": created,
	}, 1)
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title"          binding:"omitempty,max=200"`
	ReProjectID *int       `json:"re_project_id" binding:"omitempty,gt=0"`
	ContentMD   *string    `json:"content_md"`
	Priority    *int       `json:"priority"   binding:"omitempty,gte=1,lte=5"`
	Status      *string    `json:"status"     binding:"omitempty,oneof=todo done"`
	SortOrder   *int64     `json:"sort_order" binding:"omitempty,gte=0"`
	ReDueAt     *time.Time `json:"re_due_at"`
}

func (t *TaskHandler) Update(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	pidStr := c.Param("pid")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		lg.Warn("task.update.invalid_id", zap.String("id", idStr), zap.Error(err))
		utils.ReturnError(c, 4001, "非法的任务ID")
		return
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		lg.Warn("task.update.invalid_pid", zap.String("pid", pidStr), zap.Error(err))
		utils.ReturnError(c, 4001, "非法的项目ID")
		return
	}
	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("task.update.bind_failed", zap.Error(err))
		utils.ReturnError(c, 4001, "请求参数错误")
		return
	}
	in := service.UpdateTaskInput{
		Title:     req.Title,
		ProjectID: req.ReProjectID,
		ContentMD: req.ContentMD,
		Priority:  req.Priority,
		Status:    req.Status,
		ReDueAt:   req.ReDueAt,
	}
	updated, err := t.svc.Update(c.Request.Context(), lg, uid, pid, id, in)

	if updated.Affected == 0 {
		lg.Info("task.update.no_rows_affected", zap.Int("task_id", id))
		utils.ReturnSuccess(c, 0, "未修改任何字段", updated.Task, updated.Affected)
		return
	}
	lg.Info("task.update.success", zap.Int("task_id", id), zap.Int64("affected", updated.Affected))
	utils.ReturnSuccess(c, 0, "任务更新成功", updated.Task, updated.Affected)
}

func (t *TaskHandler) Delete(c *gin.Context) {
	lg := utils.CtxLogger(c)
	idStr := c.Param("id")
	uid := c.GetInt("uid")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		lg.Warn("task.delete.invalid_id", zap.String("id", idStr), zap.Error(err))
		utils.ReturnError(c, 4001, "非法的任务ID")
		return
	}
	pidStr := strings.TrimSpace(c.Query("project_id"))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		lg.Warn("task.delete.project_id_invalid", zap.String("project_id", pidStr), zap.Error(err))
		utils.ReturnError(c, 4001, "非法的项目ID")
		return
	}
	affected, err := t.svc.Delete(c.Request.Context(), lg, uid, pid, id)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
		return
	}
	utils.ReturnSuccess(c, 0, "删除成功", gin.H{
		"id":            id,
		"task_affected": affected,
	}, 1)
}

func (t *TaskHandler) Search(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	pidStr := c.Param("pid")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("task.search.invalid_id", zap.String("id", idStr), zap.Error(err))
		utils.ReturnError(c, 4001, "非法的任务ID")
		return
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil || id <= 0 {
		lg.Warn("task.search.invalid_pid", zap.String("pid", pidStr), zap.Error(err))
		utils.ReturnError(c, 4001, "非法的项目ID")
		return
	}
	task, err := t.svc.Search(c.Request.Context(), lg, id, uid, pid)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
		return
	}
	lg.Info("task.search.success", zap.Int("task_id", task.ID))
	utils.ReturnSuccess(c, 0, "查找成功", task, 1)
}

func (t *TaskHandler) List(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.DefaultQuery("status", "")
	status = strings.TrimSpace(status)
	pidStr := strings.TrimSpace(c.Query("project_id"))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		lg.Warn("task.list.project_id_invalid", zap.String("project_id", pidStr), zap.Error(err))
		utils.ReturnError(c, 4001, "非法的项目ID")
		return
	}
	in := service.TaskListInput{
		Page:   page,
		Size:   size,
		Status: status,
		Pid:    pid,
	}
	res, err := t.svc.List(c.Request.Context(), lg, uid, in)

	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
		return
	}
	lg.Info("task.list.success", zap.Int("count", len(res.Tasks)), zap.Int64("total", res.Total))
	utils.ReturnSuccess(c, 0, "获取成功", gin.H{
		"list":      res.Tasks,
		"page":      page,
		"page_size": size,
		"total":     res.Total,
	}, res.Total)

}
