package handler

import (
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

type CreateReq struct {
	Name  string  `json:"name"  binding:"required,min=1,max=128"`
	Color *string `json:"color,omitempty" binding:"omitempty,max=16"`
}

type UpdateReq struct {
	Name      *string `json:"name"  binding:"omitempty,min=1,max=128"`
	Color     *string `json:"color,omitempty" binding:"omitempty,max=16"`
	SortOrder *int64  `json:"sort_order,omitempty" binding:"omitempty"`
}

func (P *ProjectHandler) GetProjectByID(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	idStr := c.Param("id")

	profile, err := P.svc.GetProjectByID(lg, uid, idStr)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
	}
	lg.Info("project.search.ok", zap.Int("project_id", profile.ID))
	utils.ReturnSuccess(c, 0, "获取成功", gin.H{
		"project": profile,
	}, 1)
}

func (P *ProjectHandler) Search(c *gin.Context) {
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
	pslist, total, err := P.svc.GetProjectListByName(c.Request.Context(), lg, uid, name, page, size)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 5001, "系统错误")
		}
	}

	lg.Info("project.list.ok", zap.Int64("total", total), zap.Int("count", len(pslist)))
	utils.ReturnSuccess(c, 0, "获取成功", gin.H{
		"list":      pslist,
		"page":      page,
		"page_size": size,
		"total":     total,
	}, total)
}
