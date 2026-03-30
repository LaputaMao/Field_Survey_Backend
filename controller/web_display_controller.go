package controller

import (
	"net/http"
	"strconv"

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
