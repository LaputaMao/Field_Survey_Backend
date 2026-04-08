package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// GetPointListHandler 获取报表分页列表
func GetPointListHandler(c *gin.Context) {
	// 获取当前用户信息
	currentUserID, _ := c.Get("userID")
	currentUserRole, _ := c.Get("role")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	username := c.Query("username")
	dateStr := c.Query("date")
	typeStr := c.Query("type")

	items, total, err := service.GetPaginatedPoints(page, pageSize, username, dateStr, typeStr, currentUserID.(uint), currentUserRole.(string))
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

// GetPointPropertiesHandler 按 ID 获取表单明细
func GetPointPropertiesHandler(c *gin.Context) {
	// 获取当前用户信息
	currentUserID, _ := c.Get("userID")
	currentUserRole, _ := c.Get("role")

	idStr := c.Param("id")
	pointID, _ := strconv.Atoi(idStr)

	// 根据角色验证点位访问权限
	hasAccess, err := checkPointAccess(uint(pointID), currentUserID.(uint), currentUserRole.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限验证失败"})
		return
	}
	if !hasAccess {
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

// checkPointAccess 检查用户是否有权访问点位
func checkPointAccess(pointID, userID uint, userRole string) (bool, error) {
	switch userRole {
	case "first_admin":
		return true, nil
	case "sec_admin":
		// 二级管理员只能访问自己创建的项目下的点位
		var count int64
		err := config.DB.Table("points p").
			Joins("JOIN tasks t ON p.task_id = t.id").
			Joins("JOIN workspaces w ON t.workspace_id = w.id").
			Joins("JOIN projects pr ON w.project_id = pr.id").
			Where("p.id = ? AND pr.creator_id = ?", pointID, userID).
			Count(&count).Error
		return count > 0, err
	case "third_admin":
		// 三级管理员只能访问自己负责的工作区下的点位
		var count int64
		err := config.DB.Table("points p").
			Joins("JOIN tasks t ON p.task_id = t.id").
			Joins("JOIN workspaces w ON t.workspace_id = w.id").
			Where("p.id = ? AND w.assignee_id = ?", pointID, userID).
			Count(&count).Error
		return count > 0, err
	default:
		// 调查员只能访问自己提交的点位
		var count int64
		err := config.DB.Table("points").
			Where("id = ? AND user_id = ?", pointID, userID).
			Count(&count).Error
		return count > 0, err
	}
}

// GetSurveyorPointPropertiesHandler 调查员获取属性详情
func GetSurveyorPointPropertiesHandler(c *gin.Context) {
	// 从 JWT Token 解析出来的当前登录调查员 ID
	userID, _ := c.Get("userID")

	idStr := c.Param("id")
	pointID, _ := strconv.Atoi(idStr)

	props, err := service.GetSurveyorPointPropertiesById(uint(pointID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取属性详情成功",
		"data":    props,
	})
}
