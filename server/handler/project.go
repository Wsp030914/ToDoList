package handler

import (
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

type CreateReq struct {
	Name  string  `json:"name"  binding:"required,min=1,max=128"`
	Color *string `json:"color" binding:"omitempty"`
}

type UpdateReq struct {
	Name      *string `json:"name"  binding:"omitempty,min=1,max=128"`
	Color     *string `json:"color" binding:"omitempty,max=16"`
	SortOrder *int64  `json:"sort_order" binding:"omitempty"`
}

func (p *ProjectHandler) GetProjectByID(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")

	profile, err := p.svc.GetProjectByID(c.Request.Context(), lg, uid, idStr)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
		return
	}
	lg.Info("project.search.ok", zap.Int("project_id", profile.ID))
	utils.ReturnSuccess(c, 0, "获取成功", gin.H{
		"project": profile,
	}, 1)
}

func (p *ProjectHandler) Search(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	name := c.DefaultQuery("name", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	pslist, total, err := p.svc.GetProjectListByName(c.Request.Context(), lg, uid, name, page, size)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
		return
	}

	lg.Info("project.list.ok", zap.Int64("total", total), zap.Int("count", len(pslist)))
	utils.ReturnSuccess(c, 0, "获取成功", gin.H{
		"list":      pslist,
		"page":      page,
		"page_size": size,
		"total":     total,
	}, total)
}

func (p *ProjectHandler) Create(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")

	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("bind create req failed", zap.Error(err))
		utils.ReturnError(c, 4001, "参数格式错误："+err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		lg.Warn("empty project name")
		utils.ReturnError(c, 4001, "项目名称不可全空")
		return
	}
	res, err := p.svc.CreateProject(c.Request.Context(), lg, uid, name, req.Color)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
			return
		}
		lg.Error("project.CreateProject.unexpected_error", zap.Error(err))
		utils.ReturnError(c, 5001, "保存失败，请联系管理员")
		return
	}
	utils.ReturnSuccess(c, 0, "获取成功", gin.H{
		"project": res.Project,
	}, 1)
}

func (p *ProjectHandler) Update(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	lg = lg.With(zap.Int("uid", uid), zap.String("project_id", idStr))
	var req UpdateReq
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("bind update req failed", zap.Error(err))
		utils.ReturnError(c, 4001, "非法的项目ID")
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("invalid project id")
		utils.ReturnError(c, 4001, "参数格式错误："+err.Error())
		return
	}

	in := service.UpdateProjectInput{
		Name:      req.Name,
		Color:     req.Color,
		SortOrder: req.SortOrder,
	}
	updated, err := p.svc.UpdateProject(c.Request.Context(), lg, id, uid, in)

	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
		return
	}

	utils.ReturnSuccess(c, 0, "项目信息已更新", gin.H{
		"project": updated.Project,
	}, updated.Affected)
}

func (p *ProjectHandler) Delete(c *gin.Context) {
	lg := utils.CtxLogger(c)
	idStr := c.Param("id")
	uid := c.GetInt("uid")
	lg = lg.With(zap.Int("uid", uid), zap.String("id_str", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("invalid project id")
		utils.ReturnError(c, 4001, "非法的项目ID")
		return
	}

	affected, err := p.svc.DeleteProject(c.Request.Context(), lg, id, uid)
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
		"task_affected": affected.TaskAffected,
		"proj_affected": affected.Affected,
	}, 1)
}
