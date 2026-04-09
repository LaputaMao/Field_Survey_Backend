package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// GetWebWorkspaceTreeHandler ========= 接口1：Web端工作区侧边栏树路由 =========
func GetWebWorkspaceTreeHandler(c *gin.Context) {
	thirdAdminID, _ := c.Get("userID")

	results, err := service.GetWorkspaceSurveyorTree(thirdAdminID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取项目结构成功",
		"data":    results,
	})
}

// GetWebTaskDetailHandler ========= 接口2：Web端大屏级联数据详情路由 =========
func GetWebTaskDetailHandler(c *gin.Context) {
	thirdAdminID, _ := c.Get("userID")

	taskIDStr := c.Query("task_id")
	userIDStr := c.Query("user_id")

	taskID, _ := strconv.Atoi(taskIDStr)
	userID, _ := strconv.Atoi(userIDStr)

	if taskID == 0 || userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效：需要明确的 task_id 和 user_id"})
		return
	}

	data, err := service.GeTaskComprehensiveDetail(thirdAdminID.(uint), uint(taskID), uint(userID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

// GetThirdAdminWorkspaceDashboardHandler 三级管理员获取工作区仪表板数据
func GetThirdAdminWorkspaceDashboardHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "third_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅三级管理员可访问"})
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

// GetThirdAdminWorkspaceGeoJSONHandler 三级管理员获取工作区GeoJSON
func GetThirdAdminWorkspaceGeoJSONHandler(c *gin.Context) {
	thirdAdminID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "third_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅三级管理员可访问"})
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

	// 验证工作区归属：三级管理员只能访问自己负责的工作区
	var workspace model.Workspace
	if err := config.DB.First(&workspace, workspaceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作区不存在"})
		return
	}
	if workspace.AssigneeID != thirdAdminID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该工作区"})
		return
	}

	// 硬编码的全国三级生态区SHP路径，可根据需要改为前端传递 shp_path 参数
	shpPath := "./uploads/basic/全国三级生态区/全国三级生态区2024.shp"

	geoJSON, err := service.GetWorkspaceGeoJSONByWorkspaceID(uint(workspaceID), shpPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取工作区GeoJSON成功",
		"data":    geoJSON,
	})
}
