package controllers

import (
	"NewStudent/models"
	"github.com/gin-gonic/gin"
	"strconv"
)

type ProjectController struct{}

func (P ProjectController) Search(c *gin.Context) {
	uid := c.GetInt("uid")
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		ReturnError(c, 4001, "非法的项目ID")
		return
	}
	project, err := models.GetProjectInfoById(id, uid)
	if err != nil {
		ReturnError(c, 4001, "项目不存在")
		return
	}
	ReturnSuccess(c, 0, "获取成功", gin.H{
		"project": project,
	}, 1)
}

func (P ProjectController) Delete(c *gin.Context) {

	idStr := c.Param("id")
	uid := c.GetInt("uid")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		ReturnError(c, 4001, "非法的项目ID")
		return
	}
	affected, err := models.DeleteById(id, uid)
	if err != nil {
		ReturnError(c, 4001, "删除失败")
		return
	}
	if affected == 0 {

		ReturnError(c, 4001, "项目不存在或已删除")
		return
	}
	ReturnSuccess(c, 0, "删除成功", gin.H{
		"id": id,
	}, 1)
}

func (P ProjectController) Create(c *gin.Context) {

	uid := c.GetInt("uid")

	name := c.DefaultPostForm("name", "")
	if name == "" {
		ReturnError(c, 4001, "请输入项目名称")
		return
	}
	project, err := models.GetProjectInfoByNameAndUserID(name, uid)
	if project.ID != 0 {
		ReturnError(c, 4001, "该项目已存在")
		return
	}
	project, err = models.AddProject(name, uid)
	if err != nil {
		ReturnError(c, 4001, "创建失败，请联系管理员")
		return
	}
	ReturnSuccess(c, 0, "创建成功", gin.H{
		"id":         project.ID,
		"name":       project.Name,
		"sort_order": project.SortOrder,
	}, 1)
}

func (P ProjectController) List(c *gin.Context) {
	uid := c.GetInt("uid")
	if uid <= 0 {
		ReturnError(c, 401, "未授权")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	projects, total, err := models.ProjectList(uid, page, size)
	if err != nil {
		ReturnError(c, 4001, "获取项目列表信息出错")
		return
	}

	ReturnSuccess(c, 0, "获取成功", gin.H{
		"list":      projects,
		"page":      page,
		"page_size": size,
		"total":     total,
	}, total)
}
