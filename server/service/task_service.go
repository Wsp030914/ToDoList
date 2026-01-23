package service

import (
	"ToDoList/server/async"
	"ToDoList/server/models"
	"ToDoList/server/utils"
	"context"
	"errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
	"time"
)

const (
    dueScanInterval = time.Minute 
    dueScanWindow   = 5 * time.Minute    
    dueScanLimit    = 100
)
type TaskService struct {
	bus *async.EventBus
}

func NewTaskService(bus *async.EventBus) *TaskService {

	return &TaskService{bus: bus}
}

type CreateTaskInput struct {
	Title     string
	ProjectID int
	ContentMD *string
	Priority  *int
	Status    *string
	StartAt   *time.Time
	DueAt     *time.Time
}
type CreateTaskResult struct {
	Task models.Task
}
type TaskDetail struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	ProjectID   int        `json:"project_id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	ContentHtml string     `json:"content_html"`
	DueAt       *time.Time `json:"due_at"`
}

type TaskSummary struct {
	ID     int        `json:"id"`
	Title  string     `json:"title"`
	Status string     `json:"status"`
	DueAt  *time.Time `json:"due_at"`
}

func (t *TaskService) Create(ctx context.Context, lg *zap.Logger, uid int, in CreateTaskInput) (*CreateTaskResult, error) {
	lg.Info("task.create.begin",
		zap.Int("uid", uid),
		zap.Any("project_id", in.ProjectID),
		zap.Int("priority", getOr(in.Priority, 3)),
		zap.String("status", getOrStr(in.Status, "todo")),
		zap.Int("title_len", len(strings.TrimSpace(in.Title))),
		zap.Int("content_len", strlen(in.ContentMD)),
	)
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		lg.Warn("task.create.title_empty")
		return nil, &AppError{Code: 4001, Message: "请输入任务名称"}
	}
	if in.Priority != nil && (*in.Priority < 1 || *in.Priority > 5) {
		lg.Warn("task.create.priority_range_invalid", zap.Int("priority", *in.Priority))
		return nil, &AppError{Code: 4001, Message: "优先级范围应为 1~5"}
	}

	if in.StartAt != nil && in.DueAt != nil && (*in.DueAt).Before(*in.StartAt) {
		lg.Warn("task.create.time_order_invalid", zap.Timep("start_at", in.StartAt), zap.Timep("due_at", in.DueAt))
		return nil, &AppError{Code: 4001, Message: "截止时间不能早于开始时间"}
	}
	_, err := models.GetProjectByID(uid, in.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Warn("task.create.project_id_invalid", zap.Int("project_id", in.ProjectID))
			return nil, &AppError{Code: 4001, Message: "项目不存在"}
		}
		lg.Error("task.create.project_query_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "请稍后重试"}
	}
	exists, err := models.GetTaskByUserProjectTitle(uid, in.ProjectID, in.Title)
	if err != nil {
		lg.Error("task.create.check_unique_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "请稍后重试"}
	}
	if exists.ID != 0 {
		lg.Info("task.create.duplicate", zap.Int("exists_id", exists.ID))
		return nil, &AppError{Code: 4001, Message: "该任务已存在"}
	}

	status := "todo"
	if in.Status != nil {
		s := strings.TrimSpace(*in.Status)
		if s != "todo" && s != "done" {
			lg.Warn("task.create.status_invalid", zap.String("status", s))
			return nil, &AppError{Code: 4001, Message: "任务状态错误"}
		}
		status = s
	}
	priority := 3
	if in.Priority != nil {
		priority = *in.Priority
	}
	contented := ""
	if in.ContentMD != nil {
		contented = *in.ContentMD
	}
	contentHtml := ""
	if contentHtml, err = utils.RenderSafeHTML([]byte(contented)); err != nil {
		lg.Warn("task.create.md_render_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "markdown内容出错"}
	}
	task := models.Task{
		UserID:      uid,
		ProjectID:   in.ProjectID,
		Title:       in.Title,
		ContentMD:   contented,
		Status:      status,
		Priority:    priority,
		DueAt:       in.DueAt,
		ContentHtml: contentHtml,
	}
	created, err := models.CreateTaskByUidAndTask(uid, task)
	if err != nil {
		if errors.Is(err, models.ErrTaskExists) {
			lg.Info("task.create.duplicate_on_insert")
			return nil, &AppError{Code: 4001, Message: "该任务已存在"}
		}
		lg.Error("task.create.insert_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "创建失败，请稍后重试"}
	}
	return &CreateTaskResult{Task: created}, nil
}

type UpdateTaskInput struct {
	Title     *string
	ProjectID *int
	ContentMD *string
	Priority  *int
	Status    *string
	SortOrder *int64
	ReDueAt   *time.Time
}
type UpdateTaskResult struct {
	Task     models.Task
	Affected int64
}

func (t *TaskService) Update(ctx context.Context, lg *zap.Logger, uid, pid int, id int, in UpdateTaskInput) (*UpdateTaskResult, error) {
	_, err := models.GetTaskByIDAndProjectIDAndUID(id, uid, pid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.update.not_found", zap.Int("task_id", id))
			return nil, &AppError{Code: 4001, Message: "任务不存在"}
		}
		lg.Error("task.update.query_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "请稍后重试"}
	}

	update := map[string]interface{}{}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			lg.Warn("task.update.title_empty")
			return nil, &AppError{Code: 4001, Message: "任务名称不能为空"}
		}
		update["title"] = title
	}
	if in.ContentMD != nil {
		update["content_md"] = *in.ContentMD

		contentHtml := ""
		if contentHtml, err = utils.RenderSafeHTML([]byte(*in.ContentMD)); err != nil {
			lg.Warn("task.update.md_render_failed", zap.Error(err))
			return nil, &AppError{Code: 4001, Message: "markdown内容出错"}
		}
		update["content_html"] = contentHtml
	}

	if in.Priority != nil {
		if *in.Priority < 1 || *in.Priority > 5 {
			lg.Warn("task.update.priority_invalid", zap.Int("priority", *in.Priority))
			return nil, &AppError{Code: 4001, Message: "优先级范围应为 1~5"}
		}
		update["priority"] = *in.Priority
	}

	if in.Status != nil {
		s := strings.TrimSpace(*in.Status)
		if s != "todo" && s != "done" {
			lg.Warn("task.update.status_invalid", zap.String("status", s))
			return nil, &AppError{Code: 4001, Message: "任务状态错误"}
		}
		update["status"] = s
	}

	if in.SortOrder != nil && *in.SortOrder >= 0 {
		update["sort_order"] = *in.SortOrder
	}
	if in.ReDueAt != nil {
		if in.ReDueAt.Before(time.Now()) {
			lg.Warn("task.update.time_order_invalid", zap.Any("DueAt", *in.ReDueAt))
			return nil, &AppError{Code: 4001, Message: "截止时间不能早于开始时间"}
		}
		update["due_at"] = *in.ReDueAt
	}
	if in.ProjectID != nil {
		if *in.ProjectID <= 0 {
			lg.Warn("task.update.project_id_invalid", zap.Int("re_project_id", *in.ProjectID))
			return nil, &AppError{Code: 4001, Message: "项目号不合法"}
		}
		if _, err := models.GetProjectByID(uid, *in.ProjectID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				lg.Info("task.update.project_not_found", zap.Int("re_project_id", *in.ProjectID))
				return nil, &AppError{Code: 4001, Message: "项目不存在"}
			}
			lg.Error("task.update.project_query_failed", zap.Error(err))
			return nil, &AppError{Code: 4001, Message: "请稍后重试"}
		}
		update["project_id"] = *in.ProjectID
	}
	if len(update) == 0 {
		lg.Info("task.update.noop")
		return nil, &AppError{Code: 4001, Message: "没有需要更新的字段"}
	}

	updated, affected, err := models.UpdateTaskByIDAndUID(update, id, uid)
	if err != nil {
		if errors.Is(err, models.ErrTaskExists) {
			lg.Info("task.update.duplicate_on_update")
			return nil, &AppError{Code: 4001, Message: "任务已存在"}
		}
		lg.Error("task.update.update_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "更新失败，请稍后重试"}
	}

	err = DelTaskDetailCache(ctx, uid, id)
	if err != nil {
		lg.Warn("redis.del.task_detail_failed", zap.Error(err))
	}

	err = DelTaskSummaryCache(ctx, uid, pid, "all")
	if err != nil {
		lg.Warn("redis.del.task_oldsummary_failed", zap.Error(err), zap.Int("pid", pid))
	}
	if repid, ok := update["project_id"]; ok {
		err = DelTaskSummaryCache(ctx, uid, repid.(int), "all")
		if err != nil {
			lg.Warn("redis.del.task_resummary_failed", zap.Error(err), zap.Int("pid", repid.(int)))
		}
	}
	return &UpdateTaskResult{Task: updated, Affected: affected}, nil
}
func (t *TaskService) Delete(ctx context.Context, lg *zap.Logger, uid int, pid int, id int) (int64, error) {
	lg.Info("task.delete.begin", zap.Int("uid", uid), zap.Int("task_id", id), zap.Any("project_id", pid))
	affected, err := models.DeleteByIDAndProjectIDAndUID(id, pid, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.delete.not_found", zap.Int("task_id", id))
			return 0, &AppError{Code: 4001, Message: "任务不存在或已删除"}
		}
		lg.Error("task.delete.failed", zap.Error(err))
		return 0, &AppError{Code: 4001, Message: "删除失败请稍后重试"}
	}
	err = DelTaskDetailCache(ctx, uid, id)
	if err != nil {
		lg.Warn("redis.del.task_detail_failed", zap.Error(err))
	}
	err = DelTaskSummaryCache(ctx, uid, pid, "all")
	if err != nil {
		lg.Warn("redis.del.task_summary_failed", zap.Error(err), zap.Int("pid", pid))
	}
	return affected, nil
}
func (t *TaskService) Search(ctx context.Context, lg *zap.Logger, id, uid, pid int) (*TaskDetail, error) {
	lg.Info("task.search.begin", zap.Int("uid", uid), zap.Int("task_id", id))
	//redis查询缓存
	td, err := GetTaskDetailCache(ctx, uid, id)
	if err != nil {
		lg.Warn("redis.get.task_detail_failed", zap.Error(err))
	} else {
		return td, nil
	}
	//降级查db
	task, err := models.GetTaskByIDAndProjectIDAndUID(id, uid, pid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.search.not_found", zap.Int("task_id", id))
			return nil, &AppError{Code: 4001, Message: "任务不存在"}
		}
		lg.Error("task.search.query_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "任务不存在"}
	}
	//回填redis
	td = &TaskDetail{
		ID:          task.ID,
		UserID:      task.UserID,
		ProjectID: task.ProjectID,
		Title:       task.Title,
		Status:      task.Status,
		ContentHtml: task.ContentHtml ,
		DueAt:       task.DueAt,
	}
	err = SetaskDetailCache(ctx, td)
	if err != nil {
		lg.Warn("redis.get.task_detail_failed", zap.Error(err))
	}
	return td, err
}

type TaskListInput struct {
	Page   int
	Size   int
	Status string
	Pid    int
}
type TaskListResult struct {
	Tasks []TaskSummary
	Total int64
}

func (t *TaskService) List(ctx context.Context, lg *zap.Logger, uid int, in TaskListInput) (*TaskListResult, error) {
	//查询redis缓存的当前uid和pid所属的allTask
	allts, err := GetTaskSummaryCache(ctx, uid, in.Pid, in.Status)
	if err != nil {
		lg.Warn("task.list.getcachesummary_error", zap.Int("Uid", uid), zap.Int("Pid", in.Pid))
	}
	rts, rtotal, err := PageTaskSummaries(allts.Items, in.Page, in.Size)
	if err == nil {
		return &TaskListResult{Tasks: rts, Total: rtotal}, err
	}

	//降级查询mysql
	if in.Status != "todo" && in.Status != "done" && in.Status != "" {
		lg.Warn("task.list.status_invalid", zap.String("status", in.Status))
		return nil, &AppError{Code: 4001, Message: "任务状态错误"}
	}
	if in.Page < 1 {
		in.Page = 1
	}
	if in.Size <= 0 || in.Size > 100 {
		in.Size = 20
	}

	_, err = models.GetProjectByID(uid, in.Pid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("task.list.project_not_found", zap.Int("project_id", in.Pid))
			return nil, &AppError{Code: 4001, Message: "项目不存在"}
		}
		lg.Error("task.list.project_query_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "请稍后重试"}
	}

	tasks, total, err := models.TaskListAll(uid, in.Pid, in.Status)
	if err != nil {
		lg.Error("task.list.query_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "获取任务列表信息出错"}
	}

	res := make([]TaskSummary, len(tasks))
	for i := range tasks {
		res[i] = TaskSummary{
			ID:     tasks[i].ID,
			Title:  tasks[i].Title,
			Status: tasks[i].Status,
			DueAt:  tasks[i].DueAt,
		}
	}

	ts, total, err := PageTaskSummaries(res, in.Page, in.Size)
	if err != nil {
		lg.Error("task.list.page_failed", zap.Error(err))
		return nil, &AppError{Code: 4001, Message: "获取任务列表信息出错"}
	}

	err = SetTaskSummaryCache(ctx, uid, in.Pid, in.Status, total, res)
	if err != nil {
		lg.Warn("task.list.setsummarycache_error", zap.Int("Uid", uid), zap.Int("Pid", in.Pid))
	}

	return &TaskListResult{Tasks: ts, Total: total}, err
}

func (t *TaskService) checkAndNotifyDue(ctx context.Context, lg *zap.Logger) {
	
	now := time.Now()
	from := now
	to := now.Add(dueScanWindow)
	tasks, err := models.FindDueTasks(ctx, from, to, dueScanLimit)
	if err != nil {
        lg.Error("due_watcher.find_due_tasks_failed", zap.Error(err))
        return
    }
    if len(tasks) == 0 {
        return
    }

	for _, t := range tasks{
		 affected, err := models.UpdatedDueTasks(ctx, t.ID)
        if err != nil {
            lg.Error("due_watcher.mark_notified_failed",
                zap.Int("task_id", t.ID),
                zap.Error(err))
            continue
        }
        if affected == 0 {
            continue
        }
        lg.Info("due_watcher.notify",
            zap.Int("task_id", t.ID),
            zap.Int("uid", t.UserID),
            zap.Time("due_at", *t.DueAt),
        )
	}
}
func (s *TaskService) StartDueWatcher(ctx context.Context,lg *zap.Logger) {
    go func() {
        ticker := time.NewTicker(dueScanInterval)
        defer ticker.Stop()
        for {
            select {
            case <- ctx.Done():
                lg.Info("due_watcher.stopped")
                return
            case <-ticker.C:
				ctx , cancel := context.WithTimeout(context.Background(), time.Second * 30)
                s.checkAndNotifyDue(ctx, lg)
				cancel()
            }
        }
    }()
}

func PageTaskSummaries(all []TaskSummary, page, size int) ([]TaskSummary, int64, error) {
	total := len(all)
	if total == 0 {
		return []TaskSummary{}, 0, nil
	}

	start := (page - 1) * size
	if start >= total {
		return []TaskSummary{}, int64(total), nil
	}

	end := start + size
	if end > total {
		end = total
	}

	return all[start:end], int64(total), nil
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
