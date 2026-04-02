package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// GetPointListHandler 获取报表分页列表
func GetPointListHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	username := c.Query("username")
	dateStr := c.Query("date")
	typeStr := c.Query("type")

	items, total, err := service.GetPaginatedPoints(page, pageSize, username, dateStr, typeStr)
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
	idStr := c.Param("id")
	pointID, _ := strconv.Atoi(idStr)

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
