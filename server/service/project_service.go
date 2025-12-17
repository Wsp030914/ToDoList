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

func (p *ProjectService) GetProjectByID(lg *zap.Logger, uid int, idStr string) (*ProjectProfile, error) {
	lg = lg.With(zap.Int("uid", uid), zap.String("id_str", idStr))
	lg.Info("project.GetProjectByID.begin")

	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("invalid project id")
		return nil, &AppError{Code: 4001, Message: "非法项目id"}
	}

	project, err := models.GetProjectInfoByIDAndUserID(id, uid)
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
	if !ShouldBypassProjectsCache(ctx, uid) {
		ver := GetProjectVer(ctx, uid)
		items, total, redErr := GetProjectSummaryCache(ctx, uid, name, page, size, ver)
		if redErr == nil {
			lg.Info("project.GetProjectListByName.cache_hit", zap.Int64("total", total))
			return items, total, nil
		}
	}

	Projects, total, err := models.GetProjectListByUserIDAndName(uid, name, page, size)
	if err != nil {
		lg.Error("GetProjectListByName projects failed", zap.Error(err))
		return nil, 0, &AppError{Code: 5001, Message: "获取项目列表信息出错"}
	}
	res = make([]ProjectSummary, len(Projects))
	lg.Info("project.GetProjectListByName.success", zap.Int64("total", total))
	for i := range Projects {
		res[i] = ProjectSummary{
			ID:        Projects[i].ID,
			Name:      Projects[i].Name,
			Color:     Projects[i].Color,
			UpdatedAt: Projects[i].UpdatedAt,
		}
	}

	if p.bus != nil {
		ver := GetProjectVer(ctx, uid)
		infra.Publish(p.bus, lg, "PutProjectSummaryCache", struct {
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
