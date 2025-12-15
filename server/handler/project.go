package handler

import (
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"errors"
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

	res, err := P.svc.GetProjectByID(lg, uid, idStr)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) {
			utils.ReturnError(c, ae.Code, ae.Message)
		} else {
			utils.ReturnError(c, 4001, "系统错误")
		}
	}

	lg.Info("project.search.ok", zap.Int("project_id", res.Project.ID))
	utils.ReturnSuccess(c, 0, "获取成功", gin.H{
		"project": res.Project,
	}, 1)
}

func (P *ProjectHandler) Search(c *gin.Context) {
	lg := utils.CtxLogger(c)
	uid := c.GetInt("uid")
	name := c.DefaultQuery("name", "")
	if name == "" {
		lg.Warn("Project.Search.query_failed")
		utils.ReturnError(c, 4001, "查询的项目名字不可为空")
		return
	}

}
