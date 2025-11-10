package controllers

import (
	"NewStudent/models"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"strconv"
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
type UpdateTaskRequest struct {
	Title       *string    `json:"title" binding:"omitempty,max=200"`
	ProjectID   *int       `json:"project_id" binding:"required"`
	ReProjectID *int       `json:"re_project_id,omitempty"`
	ContentMD   *string    `json:"content_md,omitempty"`
	Priority    *int       `json:"priority,omitempty"`
	Status      *string    `json:"status,omitempty"`
	ReStartAt   *time.Time `json:"re_start_at,omitempty"`
	ReDueAt     *time.Time `json:"re_due_at,omitempty"`
}

func (T TaskController) Update(c *gin.Context) {
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		ReturnError(c, 4001, "非法的任务ID")
		return
	}
	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ReturnError(c, 4001, "请求参数错误")
		return
	}
	serTask, err := models.GetTaskByIDAndProjectIDAndUID(id, req.ProjectID, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ReturnError(c, 4001, "任务不存在")
			return
		}
		ReturnError(c, 5001, "请稍后重试："+err.Error())
		return
	}

	update := map[string]interface{}{}
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		update["title"] = strings.TrimSpace(*req.Title)
	} else {
		ReturnError(c, 4001, "请输入任务名称")
		return
	}

	if req.ReProjectID != nil {
		update["project_id"] = req.ReProjectID
	}

	if req.ContentMD != nil {
		update["content_md"] = *req.ContentMD
	}

	if req.Priority != nil && (*req.Priority < 1 || *req.Priority > 5) {
		ReturnError(c, 4001, "优先级范围应为 1~5")
		return
	}
	update["priority"] = *req.Priority

	StartAt := serTask.StartAt
	DueAt := serTask.DueAt
	if req.ReStartAt != nil && req.ReDueAt != nil {
		if (*req.ReDueAt).Before(*req.ReStartAt) {
			ReturnError(c, 4001, "截止时间不能早于开始时间")
			return
		}
		update["start_at"] = *req.ReStartAt
		update["due_at"] = *req.ReDueAt

	} else if req.ReStartAt != nil {
		if (*DueAt).Before(*req.ReStartAt) {
			ReturnError(c, 4001, "开始时间不能晚于截止时间")
			return
		}
		update["start_at"] = *req.ReStartAt
	} else {
		if (*req.ReDueAt).Before(*StartAt) {
			ReturnError(c, 4001, "截止时间不能早于开始时间")
			return
		}
		update["due_at"] = *req.ReDueAt
	}

	if len(update) == 0 {
		ReturnError(c, 4001, "没有需要更新的字段")
		return
	}

	updated, err, affected := models.UpdateTaskByIDAndAndProjectUID(update, id, req.ProjectID, uid)
	if err != nil {
		if errors.Is(err, models.ErrTaskExists) {
			ReturnError(c, 4001, "任务已存在")
			return
		}
		ReturnError(c, 5001, "更新失败，请稍后重试")
		return
	}
	if affected == 0 {
		ReturnSuccess(c, 0, "未修改任何字段", updated, affected)
		return
	}
	ReturnSuccess(c, 0, "项目更新成功", updated, affected)

}

func (T TaskController) StatusDelete(c *gin.Context) {
	uid := c.GetInt("uid")
	var pidPtr *int
	pidStr := strings.TrimSpace(c.Query("project_id"))
	switch {
	case pidStr == "" || strings.EqualFold(pidStr, "null"):
		pidPtr = nil
	default:
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			ReturnError(c, 4001, "非法的项目ID")
			return
		}
		pidPtr = &pid
	}
	status := c.Query("status")
	status = strings.TrimSpace(status)
	if status != "todo" && status != "doing" && status != "done" && status != "archived" && status != "" {
		ReturnError(c, 4001, "任务状态错误")
		return
	}
	err, affected := models.DeleteByStatus(uid, pidPtr, status)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ReturnError(c, 4001, "任务不存在")
			return
		}
		ReturnError(c, 5001, "请稍后重试："+err.Error())
		return
	}
	ReturnSuccess(c, 0, "删除成功", gin.H{
		"project_id":    pidPtr,
		"task_affected": affected,
	}, 1)
}
func (T TaskController) Search(c *gin.Context) {
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		ReturnError(c, 4001, "非法的任务ID")
		return
	}
	var pidPtr *int
	pidStr := strings.TrimSpace(c.Query("project_id"))
	switch {
	case pidStr == "" || strings.EqualFold(pidStr, "null"):
		pidPtr = nil
	default:
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			ReturnError(c, 4001, "非法的项目ID")
			return
		}
		pidPtr = &pid
	}
	task, err := models.GetTaskByIDAndProjectIDAndUID(id, pidPtr, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ReturnError(c, 4001, "任务不存在")
			return
		}
		ReturnError(c, 5001, "请稍后重试："+err.Error())
		return
	}
	ReturnSuccess(c, 0, "查找成功：", task, 1)
}

func (T TaskController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	uid := c.GetInt("uid")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		ReturnError(c, 4001, "非法的任务ID")
		return
	}

	var pidPtr *int
	pidStr := strings.TrimSpace(c.Query("project_id"))
	switch {
	case pidStr == "" || strings.EqualFold(pidStr, "null"):
		pidPtr = nil
	default:
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			ReturnError(c, 4001, "非法的项目ID")
			return
		}
		pidPtr = &pid
	}

	err, affected := models.DeleteByIDAndProjectIDAndUID(id, pidPtr, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ReturnError(c, 4001, "项目不存在或已删除")
			return
		}
		ReturnError(c, 5001, "删除失败")
		return
	}

	ReturnSuccess(c, 0, "删除成功", gin.H{
		"id":            id,
		"task_affected": affected,
	}, 1)
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
	if req.StartAt != nil && req.DueAt != nil && (*req.DueAt).Before(*req.StartAt) {
		ReturnError(c, 4001, "截止时间不能早于开始时间")
		return
	}

	if req.ProjectID != nil {
		_, err := models.GetProjectByID(uid, req.ProjectID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				ReturnError(c, 4001, "项目不存在")
				return
			}
			ReturnError(c, 5001, "请稍后重试："+err.Error())
			return
		}
	}

	exists, err := models.GetTaskByUserProjectTitle(uid, req.ProjectID, req.Title)
	if err != nil {
		ReturnError(c, 5001, "请稍后重试："+err.Error())
		return
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
		ReturnError(c, 5001, "创建失败，请稍后重试")
		return
	}
	ReturnSuccess(c, 0, "创建成功", gin.H{
		"task": created,
	}, 1)

}

func (T TaskController) List(c *gin.Context) {

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.DefaultQuery("status", "")
	status = strings.TrimSpace(status)
	if status != "todo" && status != "doing" && status != "done" && status != "archived" && status != "" {
		ReturnError(c, 4001, "任务状态错误")
		return
	}
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	var pidPtr *int
	pidStr := strings.TrimSpace(c.Query("project_id"))

	switch {
	case pidStr == "" || strings.EqualFold(pidStr, "null"):
		pidPtr = nil

	default:
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			ReturnError(c, 4001, "非法的项目ID")
			return
		}
		pidPtr = &pid
	}

	uid := c.GetInt("uid")
	if pidPtr != nil {
		_, err := models.GetProjectByID(uid, pidPtr)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				ReturnError(c, 4001, "项目不存在")
				return
			}
			ReturnError(c, 5001, "请稍后重试")
			return
		}
	}

	tasks, total, err := models.TaskList(uid, pidPtr, page, size, status)

	if err != nil {
		ReturnError(c, 5001, "获取任务列表信息出错")
		return
	}

	ReturnSuccess(c, 0, "获取成功", gin.H{
		"list":      tasks,
		"page":      page,
		"page_size": size,
		"total":     total,
	}, total)
}

func getString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
