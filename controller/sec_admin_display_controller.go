package controller

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ================ 二级管理员展示接口 ================

// GetSecAdminProjectsTreeHandler 获取二级管理员管辖的项目和工作区树
func GetSecAdminProjectsTreeHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "sec_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅二级管理员可访问"})
		return
	}

	projectsTree, err := service.GetProjectsTreeForSecAdmin(creatorID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取项目树成功",
		"data":    projectsTree,
	})
}

// GetSecAdminWorkspaceDashboardHandler 获取工作区仪表板数据
func GetSecAdminWorkspaceDashboardHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "sec_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅二级管理员可访问"})
		return
	}

	workspaceIDStr := c.Query("workspace_id")
	if workspaceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 workspace_id 参数"})
		return
	}

	workspaceID, err := strconv.Atoi(workspaceIDStr)
	if err != nil || workspaceID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id 参数无效"})
		return
	}

	dashboard, err := service.GetWorkspaceDashboard(uint(workspaceID), creatorID.(uint), creatorRole.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取工作区仪表板成功",
		"data":    dashboard,
	})
}

// GetSecAdminPointsHandler 获取点位分页列表（带权限过滤）
func GetSecAdminPointsHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "sec_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅二级管理员可访问"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	username := c.Query("username")
	dateStr := c.Query("date")
	typeStr := c.Query("type")
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	workspaceID, _ := strconv.Atoi(c.Query("workspace_id"))
	taskID, _ := strconv.Atoi(c.Query("task_id"))
	userID, _ := strconv.Atoi(c.Query("user_id"))

	items, total, err := service.GetPaginatedPointsForAdmin(
		page, pageSize, username, dateStr, typeStr,
		creatorID.(uint), creatorRole.(string),
		uint(projectID), uint(workspaceID), uint(taskID), uint(userID),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取点位列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"list":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetSecAdminPointPropertiesHandler 获取点位属性详情
func GetSecAdminPointPropertiesHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "sec_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅二级管理员可访问"})
		return
	}

	idStr := c.Param("id")
	pointID, err := strconv.Atoi(idStr)
	if err != nil || pointID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "点位ID无效"})
		return
	}

	// 二级管理员只能查看自己项目下的点位属性
	// 先验证点位是否属于二级管理员管辖的项目
	var count int64
	err = config.DB.Table("points p").
		Joins("JOIN tasks t ON p.task_id = t.id").
		Joins("JOIN workspaces w ON t.workspace_id = w.id").
		Joins("JOIN projects pr ON w.project_id = pr.id").
		Where("p.id = ? AND pr.creator_id = ?", pointID, creatorID.(uint)).
		Count(&count).Error

	if err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该点位"})
		return
	}

	props, err := service.GetPointPropertiesById(uint(pointID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取属性详情成功",
		"data":    props,
	})
}
