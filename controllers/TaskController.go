package controllers

import (
	"NewStudent/log"
	"NewStudent/models"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	Title       *string    `json:"title"          binding:"omitempty,max=200"`
	ReProjectID *int       `json:"re_project_id,omitempty" binding:"omitempty,gt=0"`
	MoveToInbox *bool      `json:"move_to_inbox,omitempty"`
	ContentMD   *string    `json:"content_md,omitempty"`
	Priority    *int       `json:"priority,omitempty"   binding:"omitempty,gte=1,lte=5"`
	Status      *string    `json:"status,omitempty"     binding:"omitempty,oneof=todo doing done archived"`
	SortOrder   *int64     `json:"sort_order,omitempty" binding:"omitempty,gte=0"`
	ReStartAt   *time.Time `json:"re_start_at,omitempty"`
	ReDueAt     *time.Time `json:"re_due_at,omitempty"`
}

func (T TaskController) Update(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		lg.Warn("task.update.invalid_id", zap.String("id", idStr), zap.Error(err))
		ReturnError(c, 4001, "非法的任务ID")
		return
	}
	lg.Info("task.update.begin", zap.Int("uid", uid), zap.Int("task_id", id))

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("task.update.bind_failed", zap.Error(err))
		ReturnError(c, 4001, "请求参数错误")
		return
	}

	serTask, err := models.GetTaskByIDAndUID(id, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.update.not_found", zap.Int("task_id", id))
			ReturnError(c, 4001, "任务不存在")
			return
		}
		lg.Error("task.update.query_failed", zap.Error(err))
		ReturnError(c, 5001, "请稍后重试")
		return
	}

	update := map[string]interface{}{}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			lg.Warn("task.update.title_empty")
			ReturnError(c, 4001, "任务名称不能为空")
			return
		}
		update["title"] = title
	}

	if req.ContentMD != nil {
		update["content_md"] = *req.ContentMD

		contentHtml := ""
		if contentHtml, err = RenderSafeHTML([]byte(*req.ContentMD)); err != nil {
			lg.Warn("task.update.md_render_failed", zap.Error(err))
			ReturnError(c, 4001, "markdown内容出错")
			return
		}
		update["content_html"] = contentHtml
	}

	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 5 {
			lg.Warn("task.update.priority_invalid", zap.Int("priority", *req.Priority))
			ReturnError(c, 4001, "优先级范围应为 1~5")
			return
		}
		update["priority"] = *req.Priority
	}

	if req.Status != nil {
		s := strings.TrimSpace(*req.Status)
		if s != "todo" && s != "doing" && s != "done" && s != "archived" {
			lg.Warn("task.update.status_invalid", zap.String("status", s))
			ReturnError(c, 4001, "任务状态错误")
			return
		}
		update["status"] = s
	}

	if req.SortOrder != nil && *req.SortOrder >= 0 {
		update["sort_order"] = *req.SortOrder
	}

	startOld := serTask.StartAt
	dueOld := serTask.DueAt
	if req.ReStartAt != nil && req.ReDueAt != nil {
		if req.ReDueAt.Before(*req.ReStartAt) {
			lg.Warn("task.update.time_order_invalid", zap.Any("start", *req.ReStartAt), zap.Any("due", *req.ReDueAt))
			ReturnError(c, 4001, "截止时间不能早于开始时间")
			return
		}
		update["start_at"] = *req.ReStartAt
		update["due_at"] = *req.ReDueAt
	} else if req.ReStartAt != nil {
		if dueOld != nil && dueOld.Before(*req.ReStartAt) {
			lg.Warn("task.update.start_after_due", zap.Any("start", *req.ReStartAt), zap.Any("due_old", dueOld))
			ReturnError(c, 4001, "开始时间不能晚于截止时间")
			return
		}
		update["start_at"] = *req.ReStartAt
	} else if req.ReDueAt != nil {
		if startOld != nil && req.ReDueAt.Before(*startOld) {
			lg.Warn("task.update.due_before_start", zap.Any("due", *req.ReDueAt), zap.Any("start_old", startOld))
			ReturnError(c, 4001, "截止时间不能早于开始时间")
			return
		}
		update["due_at"] = *req.ReDueAt
	}

	if req.MoveToInbox != nil && *req.MoveToInbox && req.ReProjectID != nil {
		lg.Warn("task.update.param_conflict", zap.Bool("move_to_inbox", *req.MoveToInbox), zap.Any("re_project_id", req.ReProjectID))
		ReturnError(c, 4001, "参数冲突：move_to_inbox 与 re_project_id 不能同时提供")
		return
	}

	if req.MoveToInbox != nil && *req.MoveToInbox {

		update["project_id"] = nil
	} else if req.ReProjectID != nil {

		if *req.ReProjectID <= 0 {
			lg.Warn("task.update.project_id_invalid", zap.Int("re_project_id", *req.ReProjectID))
			ReturnError(c, 4001, "项目号不合法")
			return
		}
		if _, err := models.GetProjectByID(uid, *req.ReProjectID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				lg.Info("task.update.project_not_found", zap.Int("re_project_id", *req.ReProjectID))
				ReturnError(c, 4001, "项目不存在")
				return
			}
			lg.Error("task.update.project_query_failed", zap.Error(err))
			ReturnError(c, 5001, "请稍后重试")
			return
		}
		update["project_id"] = *req.ReProjectID
	}

	if len(update) == 0 {
		lg.Info("task.update.noop")
		ReturnError(c, 4001, "没有需要更新的字段")
		return
	}

	updated, err, affected := models.UpdateTaskByIDAndUID(update, id, uid)
	if err != nil {
		if errors.Is(err, models.ErrTaskExists) {
			lg.Info("task.update.duplicate_on_update")
			ReturnError(c, 4001, "任务已存在")
			return
		}
		lg.Error("task.update.update_failed", zap.Error(err))
		ReturnError(c, 5001, "更新失败，请稍后重试")
		return
	}
	if affected == 0 {
		lg.Info("task.update.no_rows_affected", zap.Int("task_id", id))
		ReturnSuccess(c, 0, "未修改任何字段", updated, affected)
		return
	}
	lg.Info("task.update.success", zap.Int("task_id", id), zap.Int64("affected", affected))
	ReturnSuccess(c, 0, "任务更新成功", updated, affected)
}

func (T TaskController) StatusDelete(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")

	var pidPtr *int
	pidStr := strings.TrimSpace(c.Query("project_id"))
	switch {
	case pidStr == "" || strings.EqualFold(pidStr, "null"):
		pidPtr = nil
	default:
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			lg.Warn("task.status_delete.project_id_invalid", zap.String("project_id", pidStr), zap.Error(err))
			ReturnError(c, 4001, "非法的项目ID")
			return
		}
		pidPtr = &pid
	}
	status := c.Query("status")
	status = strings.TrimSpace(status)
	if status != "todo" && status != "doing" && status != "done" && status != "archived" {
		lg.Warn("task.status_delete.status_invalid", zap.String("status", status))
		ReturnError(c, 4001, "任务状态错误")
		return
	}
	err, affected := models.DeleteByStatus(uid, pidPtr, status)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.status_delete.not_found")
			ReturnError(c, 4001, "任务不存在")
			return
		}
		lg.Error("task.status_delete.failed", zap.Error(err))
		ReturnError(c, 5001, "请稍后重试")
		return
	}
	lg.Info("task.status_delete.success", zap.Int64("affected", affected))
	ReturnSuccess(c, 0, "删除成功", gin.H{
		"project_id":    pidPtr,
		"task_affected": affected,
	}, 1)
}

func (T TaskController) Search(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("task.search.invalid_id", zap.String("id", idStr), zap.Error(err))
		ReturnError(c, 4001, "非法的任务ID")
		return
	}
	lg.Info("task.search.begin", zap.Int("uid", uid), zap.Int("task_id", id))

	task, err := models.GetTaskByIDAndUID(id, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.search.not_found", zap.Int("task_id", id))
			ReturnError(c, 4001, "任务不存在")
			return
		}
		lg.Error("task.search.query_failed", zap.Error(err))
		ReturnError(c, 5001, "请稍后重试")
		return
	}
	lg.Info("task.search.success", zap.Int("task_id", task.ID))
	ReturnSuccess(c, 0, "查找成功", task, 1)
}

func (T TaskController) Delete(c *gin.Context) {
	lg := log.CtxLogger(c)
	idStr := c.Param("id")
	uid := c.GetInt("uid")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("task.delete.invalid_id", zap.String("id", idStr), zap.Error(err))
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
			lg.Warn("task.delete.project_id_invalid", zap.String("project_id", pidStr), zap.Error(err))
			ReturnError(c, 4001, "非法的项目ID")
			return
		}
		pidPtr = &pid
	}
	lg.Info("task.delete.begin", zap.Int("uid", uid), zap.Int("task_id", id), zap.Any("project_id", pidPtr))
	err, affected := models.DeleteByIDAndProjectIDAndUID(id, pidPtr, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.delete.not_found", zap.Int("task_id", id))
			ReturnError(c, 4001, "任务不存在或已删除")
			return
		}
		lg.Error("task.delete.failed", zap.Error(err))
		ReturnError(c, 5001, "删除失败")
		return
	}
	lg.Info("task.delete.success", zap.Int64("affected", affected))
	ReturnSuccess(c, 0, "删除成功", gin.H{
		"id":            id,
		"task_affected": affected,
	}, 1)
}

func (T TaskController) Create(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("task.create.bind_failed", zap.Error(err))
		ReturnError(c, 4001, "请求参数错误")
		return
	}
	lg.Info("task.create.begin",
		zap.Int("uid", uid),
		zap.Any("project_id", req.ProjectID),
		zap.Int("priority", getOr(req.Priority, 3)),
		zap.String("status", getOrStr(req.Status, "todo")),
		zap.Int("title_len", len(strings.TrimSpace(req.Title))),
		zap.Int("content_len", strlen(req.ContentMD)),
	)

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		lg.Warn("task.create.title_empty")
		ReturnError(c, 4001, "请输入任务名称")
		return
	}
	if req.Priority != nil && (*req.Priority < 1 || *req.Priority > 5) {
		lg.Warn("task.create.priority_range_invalid", zap.Int("priority", *req.Priority))
		ReturnError(c, 4001, "优先级范围应为 1~5")
		return
	}
	if req.StartAt != nil && req.DueAt != nil && (*req.DueAt).Before(*req.StartAt) {
		lg.Warn("task.create.time_order_invalid", zap.Timep("start_at", req.StartAt), zap.Timep("due_at", req.DueAt))
		ReturnError(c, 4001, "截止时间不能早于开始时间")
		return
	}

	if req.ProjectID != nil {
		if *req.ProjectID <= 0 {
			lg.Warn("task.create.project_id_invalid", zap.Int("project_id", *req.ProjectID))
			ReturnError(c, 4001, "项目号不合法")
			return
		}
		_, err := models.GetProjectByID(uid, *req.ProjectID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				lg.Warn("task.create.project_id_invalid", zap.Int("project_id", *req.ProjectID))
				ReturnError(c, 4001, "项目不存在")
				return
			}
			lg.Error("task.create.project_query_failed", zap.Error(err))
			ReturnError(c, 5001, "请稍后重试")
			return
		}
	}

	exists, err := models.GetTaskByUserProjectTitle(uid, req.ProjectID, req.Title)
	if err != nil {
		lg.Error("task.create.check_unique_failed", zap.Error(err))
		ReturnError(c, 5001, "请稍后重试")
		return
	}
	if exists.ID != 0 {
		lg.Info("task.create.duplicate", zap.Int("exists_id", exists.ID))
		ReturnError(c, 4001, "该任务已存在")
		return
	}

	status := "todo"
	if req.Status != nil {
		s := strings.TrimSpace(*req.Status)
		lg.Warn("task.create.status_invalid", zap.String("status", s))
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
	contented := ""
	if req.ContentMD != nil {
		contented = *req.ContentMD
	}

	contentHtml := ""
	if contentHtml, err = RenderSafeHTML([]byte(contented)); err != nil {
		lg.Warn("task.create.md_render_failed", zap.Error(err))
		ReturnError(c, 4001, "markdown内容出错")
		return
	}

	task := models.Task{
		UserID:      uid,
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		ContentMD:   contented,
		Status:      status,
		Priority:    priority,
		StartAt:     req.StartAt,
		DueAt:       req.DueAt,
		ContentHtml: contentHtml,
	}

	created, err := models.CreateTaskByUidAndTask(uid, task)
	if err != nil {
		if errors.Is(err, models.ErrTaskExists) {
			lg.Info("task.create.duplicate_on_insert")
			ReturnError(c, 4001, "该任务已存在")
			return
		}
		lg.Error("task.create.insert_failed", zap.Error(err))
		ReturnError(c, 5001, "创建失败，请稍后重试")
		return
	}
	lg.Info("task.create.success", zap.Int("task_id", created.ID))
	ReturnSuccess(c, 0, "创建成功", gin.H{
		"task": created,
	}, 1)

}

func (T TaskController) List(c *gin.Context) {
	lg := log.CtxLogger(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.DefaultQuery("status", "")
	status = strings.TrimSpace(status)

	if status != "todo" && status != "doing" && status != "done" && status != "archived" && status != "" {
		lg.Warn("task.list.status_invalid", zap.String("status", status))
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
			lg.Warn("task.list.project_id_invalid", zap.String("project_id", pidStr), zap.Error(err))
			ReturnError(c, 4001, "非法的项目ID")
			return
		}
		pidPtr = &pid
	}

	uid := c.GetInt("uid")
	lg.Info("task.list.begin", zap.Int("uid", uid), zap.Any("project_id", pidPtr), zap.String("status", status), zap.Int("page", page), zap.Int("size", size))

	if pidPtr != nil {
		if *pidPtr <= 0 {
			lg.Warn("task.list.project_id_invalid_value", zap.Int("project_id", *pidPtr))
			ReturnError(c, 4001, "项目号不合法")
			return
		}
		_, err := models.GetProjectByID(uid, *pidPtr)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				lg.Info("task.list.project_not_found", zap.Int("project_id", *pidPtr))
				ReturnError(c, 4001, "项目不存在")
				return
			}
			lg.Error("task.list.project_query_failed", zap.Error(err))
			ReturnError(c, 5001, "请稍后重试")
			return
		}
	}

	tasks, total, err := models.TaskList(uid, pidPtr, page, size, status)
	if err != nil {
		lg.Error("task.list.query_failed", zap.Error(err))
		ReturnError(c, 5001, "获取任务列表信息出错")
		return
	}
	lg.Info("task.list.success", zap.Int("count", len(tasks)), zap.Int64("total", total))
	ReturnSuccess(c, 0, "获取成功", gin.H{
		"list":      tasks,
		"page":      page,
		"page_size": size,
		"total":     total,
	}, total)
}

func keys(m map[string]interface{}) []string {
	arr := make([]string, 0, len(m))
	for k := range m {
		arr = append(arr, k)
	}
	return arr
}
func strlen(p *string) int {
	if p == nil {
		return 0
	}
	return len(*p)
}
func getOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
func getOrStr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}
