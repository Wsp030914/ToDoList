package service

import (
	"ToDoList/server/async"
	"ToDoList/server/infra"
	"ToDoList/server/models"
	"context"
	"errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

type ProjectProfile struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UserID    int       `json:"user_id"`
	Color     string    `json:"color"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	SortOrder int64     `json:"sort_order"`
}
type ProjectSummary struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	UpdatedAt time.Time `json:"updated_at"`
}
type ProjectService struct {
	bus *async.EventBus
}

func NewProjectService(bus *async.EventBus) *ProjectService {
	return &ProjectService{bus: bus}
}

func (p *ProjectService) GetProjectByID(ctx context.Context, lg *zap.Logger, uid int, idStr string) (*ProjectProfile, error) {
	lg = lg.With(zap.Int("uid", uid), zap.String("id_str", idStr))
	lg.Info("project.GetProjectByID.begin")

	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("invalid project id")
		return nil, &AppError{Code: 4001, Message: "非法项目id"}
	}

	project, err := models.GetProjectInfoByIDAndUserID(ctx, id, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("project not found", zap.Int("project_id", id))
			return nil, &AppError{Code: 4001, Message: "项目不存在"}
		}
		lg.Error("query project failed", zap.Error(err))
		return nil, &AppError{Code: 5001, Message: "请稍后重试"}
	}
	lg.Info("project.GetProjectByID.success", zap.Int("project_id", project.ID))
	return &ProjectProfile{
		ID:        project.ID,
		Name:      project.Name,
		UserID:    project.UserID,
		Color:     project.Color,
		UpdatedAt: project.UpdatedAt,
		CreatedAt: project.CreatedAt,
		SortOrder: project.SortOrder,
	}, nil
}

func (p *ProjectService) GetProjectListByName(ctx context.Context, lg *zap.Logger, uid int, name string, page int, size int) ([]ProjectSummary, int64, error) {
	var res []ProjectSummary
	lg = lg.With(zap.Int("uid", uid), zap.String("name", name))
	lg.Info("project.GetProjectListByName.begin")

	useCache := !ShouldBypassProjectsCache(ctx, uid)
	var ver int64
	if useCache {
		ver = GetProjectsVer(ctx, uid)

		items, total, redErr := GetProjectsSummaryCache(ctx, uid, name, page, size, ver)
		if redErr == nil {
			lg.Info("project.GetProjectListByName.cache_hit", zap.Int64("total", total))
			return items, total, nil
		}
	}

	Projects, total, err := models.GetProjectListByUserIDAndName(ctx, uid, name, page, size)
	if err != nil {
		lg.Error("GetProjectListByName projects failed", zap.Error(err))
		return nil, 0, &AppError{Code: 5001, Message: "获取项目列表信息出错"}
	}

	res = make([]ProjectSummary, len(Projects))
	for i := range Projects {
		res[i] = ProjectSummary{
			ID:        Projects[i].ID,
			Name:      Projects[i].Name,
			Color:     Projects[i].Color,
			UpdatedAt: Projects[i].UpdatedAt,
		}
	}

	if useCache && p.bus != nil {
		infra.Publish(p.bus, lg, "PutProjectsSummaryCache", struct {
			Items []ProjectSummary `json:"items"`
			Total int64            `json:"total"`
			UID   int              `json:"uid"`
			Ver   int64            `json:"ver"`
			Name  string           `json:"name"`
			Page  int              `json:"page"`
			Size  int              `json:"size"`
		}{Items: res, Total: total, UID: uid, Ver: ver, Name: name, Page: page, Size: size}, 100*time.Millisecond)
	}

	return res, total, nil
}


type CreateResult struct {
	Project models.Project
}

func (p *ProjectService) CreateProject(ctx context.Context, lg *zap.Logger, uid int, name string, color *string) (*CreateResult, error) {
	lg = lg.With(zap.Int("uid", uid), zap.String("name", name))
	lg.Info("project.CreateProject.begin")
	var project models.Project
	if color != nil {
		cl := strings.TrimSpace(*color)
		project = models.Project{Name: name, UserID: uid, Color: cl}
	} else {
		project = models.Project{Name: name, UserID: uid}
	}
	created, err := models.AddProject(ctx, project)
	if err != nil {
		if errors.Is(err, models.ErrProjectExists) {
			lg.Info("CreateProject.duplicate_on_insert")
			return nil, &AppError{Code: 4001, Message: "该项目已存在"}
		}
		lg.Error("CreateProject_failed", zap.Error(err))
		return nil, &AppError{Code: 5001, Message: "保存失败，请联系管理员"}
	}
	err = IncrProjectsVer(ctx, c.Rdb, uid)
	if err != nil {
		lg.Warn("project.CreateProject.incr_ver_failed", zap.Error(err))
	}

	lg.Info("project.CreateProject.success", zap.Int("project_id", created.ID))
	return &CreateResult{
		Project: created,
	}, nil
}

type UpdateProjectInput struct {
	Name      *string `json:"name"  binding:"omitempty,min=1,max=128"`
	Color     *string `json:"color" binding:"omitempty,max=16"`
	SortOrder *int64  `json:"sort_order" binding:"omitempty"`
}
type UpdateProjectResult struct {
	Project  models.Project
	Affected int64
}

func (p *ProjectService) UpdateProject(ctx context.Context, lg *zap.Logger, pid int, uid int, in UpdateProjectInput) (*UpdateProjectResult, error) {
	update := map[string]interface{}{}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		name := strings.TrimSpace(*in.Name)
		update["name"] = name
	}
	if in.Color != nil {
		Color := strings.TrimSpace(*in.Color)
		update["color"] = Color
	}
	if in.SortOrder != nil {
		if *in.SortOrder < 0 {
			return nil, &AppError{Code: 4001, Message: "sort_order 不能小于 0"}
		}
		update["sort_order"] = *in.SortOrder
	}
	if len(update) == 0 {
		lg.Info("project.no fields to update")
		return nil, &AppError{Code: 4001, Message: "没有需要更新的字段"}
	}

	updated, affected, err := models.UpdateProjectByIDAndUserID(ctx, update, pid, uid)
	if err != nil {
		if errors.Is(err, models.ErrProjectExists) {
			lg.Info("project.UpdateProject.duplicate_name", zap.Int("project_id", pid))
			return nil, &AppError{Code: 4001, Message: "该项目已存在"}
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("project.UpdateProject.not_found", zap.Int("project_id", pid))
			return nil, &AppError{Code: 4004, Message: "项目不存在或无权限"}
		}
		lg.Error("project.UpdateProject.db_failed", zap.Int("project_id", pid), zap.Error(err))
		return nil, &AppError{Code: 5001, Message: "保存失败，请联系管理员"}
	}
	if affected == 0 {
		lg.Info("Project.update.noop")
		return &UpdateProjectResult{
			Project:  updated,
			Affected: affected,
		}, nil
	}
	err = IncrProjectsVer(ctx, c.Rdb, uid)
	if err != nil {
		lg.Warn("project.UpdateProject.incr_ver_failed", zap.Error(err))
	}
	lg.Info("project.update.ok", zap.Int("project_id", updated.ID), zap.Int64("affected", affected))
	return &UpdateProjectResult{
		Project:  updated,
		Affected: affected,
	}, nil
}

type DeleteProjectResult struct {
	Affected     int64
	TaskAffected int64
}

func (p *ProjectService) DeleteProject(ctx context.Context, lg *zap.Logger, pid int, uid int) (*DeleteProjectResult, error) {
	affected, taskAffected, err := models.DeleteProjectAndTasks(ctx, pid, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("project not found or already deleted", zap.Int("project_id", pid))
			return nil, &AppError{Code: 4001, Message: "项目不存在或已删除"}
		}
		lg.Error("delete project failed", zap.Error(err))
		return nil, &AppError{Code: 5001, Message: "删除失败"}
	}
	lg.Info("project.delete.ok",
		zap.Int("project_id", pid),
		zap.Int64("proj_affected", affected),
		zap.Int64("task_affected", taskAffected),
	)

	return &DeleteProjectResult{
		Affected:     affected,
		TaskAffected: taskAffected,
	}, nil
}
