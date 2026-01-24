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

// @Summary 获取项目详情
// @Description 根据项目ID获取项目的详细信息，包括名称、颜色和创建时间
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path integer true "项目ID"
// @Success 200 {object} ProjectDetailResponse "获取成功，返回项目信息"
// @Failure 400 {object} ErrorResponse "非法的项目ID"
// @Failure 401 {object} ErrorResponse "未授权或token无效"
// @Failure 404 {object} ErrorResponse "项目不存在"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /projects/{id} [get]
func (p *ProjectHandler) GetProjectByID(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	start := time.Now()
	lg = lg.With(zap.Int("uid", uid), zap.String("id_Str", idStr))
	profile, err := p.svc.GetProjectByID(c.Request.Context(), lg, uid, idStr)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			lg.Warn("project.search.failed", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Warn("project.search.error", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}
	lg.Info("project.search.ok", zap.Int("project_id", profile.ID), zap.Duration("elapsed_ms", time.Since(start)))
	utils.ReturnSuccess(c, utils.CodeOK, "获取成功", gin.H{
		"project": profile,
	}, 1)
}

// @Summary 搜索项目列表
// @Description 获取当前用户的项目列表，支持按名称模糊搜索和分页
// @Accept json
// @Produce json
// @Security Bearer
// @Param name query string false "项目名称关键字（模糊搜索）"
// @Param page query integer false "页码（默认1）"
// @Param page_size query integer false "每页数量（默认20，最大100）"
// @Success 200 {object} ProjectListResponse "获取成功，返回项目列表"
// @Failure 401 {object} ErrorResponse "未授权或token无效"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /projects [get]
func (p *ProjectHandler) Search(c *gin.Context) {
	lg := utils.CtxLogger(c)
	start := time.Now()
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
	lg = lg.With(zap.Int("uid", uid), zap.String("name", name))
	pslist, total, err := p.svc.SearchProjectListByName(c.Request.Context(), lg, uid, name, page, size)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			lg.Warn("project.list.failed", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Error("project.list.error", zap.Error(err), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}
	lg.Info("project.list.ok", zap.Int64("total", total), zap.Int("count", len(pslist)), zap.Duration("elapsed_ms", time.Since(start)))
	utils.ReturnSuccess(c, utils.CodeOK, "获取成功", gin.H{
		"list":      pslist,
		"page":      page,
		"page_size": size,
		"total":     total,
	}, total)
}

// @Summary 创建项目
// @Description 创建新项目，项目名称必需且不能为空
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body CreateReq true "项目创建请求体"
// @Success 200 {object} ProjectCreateResponse "创建成功，返回项目信息"
// @Failure 400 {object} ErrorResponse "参数错误"
// @Failure 401 {object} ErrorResponse "未授权或token无效"
// @Failure 409 {object} ErrorResponse "项目重复"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /projects [post]
func (p *ProjectHandler) Create(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	start := time.Now()
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("project.create.param_bind_failed", zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "参数格式错误："+err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		lg.Warn("project.create.empty_name")
		utils.ReturnError(c, utils.ErrCodeValidation, "项目名称不可为空")
		return
	}
	lg = lg.With(zap.Int("uid", uid), zap.String("name", name))

	res, err := p.svc.CreateProject(c.Request.Context(), lg, uid, name, req.Color)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			lg.Warn("project.create.failed", zap.Int("code", ae.Code), zap.Duration("elapsed_ms", time.Since(start)))
			utils.ReturnError(c, ae.Code, ae.Message)
			return
		}
		lg.Error("project.create.error", zap.Error(err), zap.Duration("elapsed_ms", time.Since(start)))
		utils.ReturnError(c, utils.ErrCodeInternalServer, "保存失败，请联系管理员")
		return
	}
	lg.Info("project.create.success", zap.Int("id", res.Project.ID), zap.Duration("elapsed_ms", time.Since(start)))
	utils.ReturnSuccess(c, utils.CodeOK, "项目创建成功", gin.H{
		"project": res.Project,
	}, 1)
}

// @Summary 更新项目信息
// @Description 更新项目的名称、颜色和排序顺序（可选项）
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path integer true "项目ID"
// @Param body body UpdateReq true "项目更新请求体"
// @Success 200 {object} ProjectUpdateResponse "更新成功，返回更新后的项目信息"
// @Failure 400 {object} ErrorResponse "非法的项目ID或参数格式错误"
// @Failure 401 {object} ErrorResponse "未授权或token无效"
// @Failure 404 {object} ErrorResponse "项目不存在"
// @Failure 409 {object} ErrorResponse "项目重复"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /projects/{id} [patch]
func (p *ProjectHandler) Update(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	lg = lg.With(zap.Int("uid", uid), zap.String("project_id", idStr))
	var req UpdateReq
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("project.update.param_invalid", zap.String("id", idStr), zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "非法的项目ID")
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		lg.Warn("project.update.param_bind_failed", zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "参数格式错误："+err.Error())
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
			lg.Warn("project.update.failed", zap.Int("code", ae.Code))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Error("project.update.error", zap.Error(err))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}

	lg.Info("project.update.success", zap.Int64("affected", updated.Affected))
	utils.ReturnSuccess(c, utils.CodeOK, "项目信息已更新", gin.H{
		"project": updated.Project,
	}, updated.Affected)
}

// @Summary 删除项目
// @Description 删除项目及其关联的所有任务
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path integer true "项目ID"
// @Success 200 {object} ProjectDeleteResponse "删除成功，返回删除的项目ID和受影响的行数"
// @Failure 400 {object} ErrorResponse "非法的项目ID"
// @Failure 401 {object} ErrorResponse "未授权或token无效"
// @Failure 404 {object} ErrorResponse "项目不存在"
// @Failure 500 {object} ErrorResponse "系统错误"
// @Router /projects/{id} [delete]
func (p *ProjectHandler) Delete(c *gin.Context) {
	lg := utils.CtxLogger(c)
	idStr := c.Param("id")
	uid := c.GetInt("uid")
	lg = lg.With(zap.Int("uid", uid), zap.String("id_str", idStr))
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		lg.Warn("project.delete.param_invalid", zap.String("id", idStr), zap.Error(err))
		utils.ReturnError(c, utils.ErrCodeValidation, "非法的项目ID")
		return
	}

	affected, err := p.svc.DeleteProject(c.Request.Context(), lg, id, uid)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			lg.Warn("project.delete.failed", zap.Int("code", ae.Code))
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			lg.Error("project.delete.error", zap.Error(err))
			utils.ReturnError(c, utils.ErrCodeInternalServer, "系统错误")
		}
		return
	}

	lg.Info("project.delete.success", zap.Int("id", id), zap.Int64("proj_affected", affected.Affected), zap.Int64("task_affected", affected.TaskAffected))
	utils.ReturnSuccess(c, utils.CodeOK, "删除成功", gin.H{
		"id":            id,
		"task_affected": affected.TaskAffected,
		"proj_affected": affected.Affected,
	}, 1)
}
