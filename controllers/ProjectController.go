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
)

type ProjectController struct{}

type CreateReq struct {
	Name  string  `json:"name"  binding:"required,min=1,max=128"`
	Color *string `json:"color,omitempty" binding:"omitempty,max=16"`
}
type UpdateReq struct {
	Name      *string `json:"name"  binding:"omitempty,min=1,max=128"`
	Color     *string `json:"color,omitempty" binding:"omitempty,max=16"`
	SortOrder *int64  `json:"sort_order,omitempty" binding:"omitempty"`
}

func (P ProjectController) Search(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")

	lg = lg.With(zap.Int("uid", uid), zap.String("id_str", idStr))
	lg.Info("project.search.begin")

	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("invalid project id")
		ReturnError(c, 4001, "非法的项目ID")
		return
	}

	project, err := models.GetProjectInfoByIDAndUserID(id, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("project not found", zap.Int("project_id", id))
			ReturnError(c, 4001, "项目不存在")
			return
		}
		lg.Error("query project failed", zap.Error(err))
		ReturnError(c, 5001, "请稍后重试")
		return
	}

	lg.Info("project.search.ok", zap.Int("project_id", project.ID))
	ReturnSuccess(c, 0, "获取成功", gin.H{
		"project": project,
	}, 1)
}

func (P ProjectController) Delete(c *gin.Context) {
	lg := log.CtxLogger(c)
	idStr := c.Param("id")
	uid := c.GetInt("uid")
	lg = lg.With(zap.Int("uid", uid), zap.String("id_str", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("invalid project id")
		ReturnError(c, 4001, "非法的项目ID")
		return
	}

	projAffected, taskAffected, err := models.DeleteProjectAndTasks(id, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("project not found or already deleted", zap.Int("project_id", id))
			ReturnError(c, 4001, "项目不存在或已删除")
			return
		}
		lg.Error("delete project failed", zap.Error(err))
		ReturnError(c, 5001, "删除失败")
		return
	}

	lg.Info("project.delete.ok",
		zap.Int("project_id", id),
		zap.Int64("proj_affected", projAffected),
		zap.Int64("task_affected", taskAffected),
	)
	ReturnSuccess(c, 0, "删除成功", gin.H{
		"id":            id,
		"task_affected": taskAffected,
		"proj_affected": projAffected,
	}, 1)
}

func (P ProjectController) Create(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")

	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("bind create req failed", zap.Error(err))
		ReturnError(c, 4001, "参数格式错误："+err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		lg.Warn("empty project name")
		ReturnError(c, 4001, "项目名称不可全空")
		return
	}

	project := models.Project{Name: name, UserID: uid}
	if req.Color != nil {
		color := strings.TrimSpace(*req.Color)
		project.Color = color
	}
	created, err := models.AddProject(project)
	if err != nil {
		if errors.Is(err, models.ErrProjectExists) {
			lg.Info("project already exists", zap.String("name", name))
			ReturnError(c, 4001, "该项目已存在")
			return
		}
		lg.Error("create project failed", zap.Error(err))
		ReturnError(c, 5001, "创建失败，请联系管理员")
		return
	}
	lg.Info("project.create.ok",
		zap.Int("project_id", created.ID),
		zap.String("name", created.Name),
		zap.Int64("sort_order", created.SortOrder),
		zap.String("color", created.Color),
	)
	ReturnSuccess(c, 0, "创建成功", gin.H{
		"id":         created.ID,
		"name":       created.Name,
		"sort_order": created.SortOrder,
		"color":      created.Color,
	}, 1)
}

func (P ProjectController) List(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	lg = lg.With(zap.Int("uid", uid), zap.Int("page", page), zap.Int("size", size))
	lg.Info("project.list.begin")

	projects, total, err := models.ProjectList(uid, page, size)
	if err != nil {
		lg.Error("list projects failed", zap.Error(err))
		ReturnError(c, 5001, "获取项目列表信息出错")
		return
	}

	lg.Info("project.list.ok", zap.Int64("total", total), zap.Int("count", len(projects)))
	ReturnSuccess(c, 0, "获取成功", gin.H{
		"list":      projects,
		"page":      page,
		"page_size": size,
		"total":     total,
	}, total)
}

func (P ProjectController) Update(c *gin.Context) {
	lg := log.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	lg = lg.With(zap.Int("uid", uid), zap.String("id_str", idStr))

	var req UpdateReq
	update := map[string]interface{}{}
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("bind update req failed", zap.Error(err))
		ReturnError(c, 4001, "非法的项目ID")
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("invalid project id")
		ReturnError(c, 4001, "参数格式错误："+err.Error())
		return
	}
	_, err = models.GetProjectInfoByIDAndUserID(id, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			lg.Info("project not found", zap.Int("project_id", id))
			ReturnError(c, 4001, "项目不存在")
			return
		}
		lg.Error("query project failed", zap.Error(err))
		ReturnError(c, 5001, "请稍后重试")
		return
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		name := strings.TrimSpace(*req.Name)
		update["name"] = name
	}
	if req.Color != nil {
		Color := strings.TrimSpace(*req.Color)
		update["color"] = Color
	}
	if req.SortOrder != nil && *req.SortOrder >= 0 {
		SortOrder := *req.SortOrder
		update["sort_order"] = SortOrder
	}
	if len(update) == 0 {
		lg.Info("no fields to update")
		ReturnError(c, 4001, "没有需要更新的字段")
		return
	}
	updated, err, affected := models.UpdateProjectByIDAndUserID(update, id, uid)
	if err != nil {
		if errors.Is(err, models.ErrProjectExists) {
			lg.Info("duplicate project name on update", zap.Any("update", update))
			ReturnError(c, 4001, "项目已存在")
			return
		}
		lg.Error("update project failed", zap.Error(err), zap.Any("update", update))
		ReturnError(c, 5001, "更新失败，请稍后重试")
		return
	}
	if affected == 0 {
		lg.Info("project.update.noop", zap.Int("project_id", id))
		ReturnSuccess(c, 0, "未修改任何字段", updated, affected)
		return
	}
	lg.Info("project.update.ok", zap.Int("project_id", id), zap.Int64("affected", affected))
	ReturnSuccess(c, 0, "项目更新成功", updated, affected)

}
