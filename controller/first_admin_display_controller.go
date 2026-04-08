package controller

import (
	"Field_Survey_Backend/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ================ 一级管理员展示接口 ================

// GetFirstAdminProjectsTreeHandler 获取所有项目和工作区树
func GetFirstAdminProjectsTreeHandler(c *gin.Context) {
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "first_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅一级管理员可访问"})
		return
	}

	projectsTree, err := service.GetProjectsTreeForFirstAdmin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取项目树成功",
		"data":    projectsTree,
	})
}

// GetFirstAdminWorkspaceDashboardHandler 获取工作区仪表板数据
func GetFirstAdminWorkspaceDashboardHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "first_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅一级管理员可访问"})
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

// GetFirstAdminPointsHandler 获取点位分页列表（带权限过滤）
func GetFirstAdminPointsHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "first_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅一级管理员可访问"})
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

// GetFirstAdminPointPropertiesHandler 获取点位属性详情
func GetFirstAdminPointPropertiesHandler(c *gin.Context) {
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "first_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅一级管理员可访问"})
		return
	}

	idStr := c.Param("id")
	pointID, err := strconv.Atoi(idStr)
	if err != nil || pointID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "点位ID无效"})
		return
	}

	// 一级管理员可以直接查看任何点位属性
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
