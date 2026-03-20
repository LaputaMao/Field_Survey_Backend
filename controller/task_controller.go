package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// ---- [三管专属] ----

// BulkAssignTaskHandler 三管批量上传
func BulkAssignTaskHandler(c *gin.Context) {
	thirdAdminID, _ := c.Get("userID")

	workspaceIDStr := c.PostForm("workspace_id")
	workspaceID, _ := strconv.Atoi(workspaceIDStr)

	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传包含线路和点位 .shp 文件的 ZIP 压缩包"})
		return
	}
	defer file.Close()

	// 触发全自动批量解析派发
	err = service.BulkAssignTasks(uint(workspaceID), thirdAdminID.(uint), file, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "批量任务解析拆分并派发成功",
		//"create_count": successCount,
	})
}

// ---- [野外调查员专属] ----

func GetSurveyorTasksHandler(c *gin.Context) {
	surveyorID, _ := c.Get("userID")
	tasks, err := service.GetSurveyorTasks(surveyorID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务列表失败"})
		return
	}

	unreadCount := 0
	for _, t := range tasks {
		if !t.IsRead {
			unreadCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": tasks, "unread_count": unreadCount})
}

func ReadTaskAndGetGeoJSONHandler(c *gin.Context) {
	surveyorID, _ := c.Get("userID")
	taskIDStr := c.Param("id")
	taskID, _ := strconv.Atoi(taskIDStr)

	// 1. 消除未读小红点
	_ = service.MarkTaskAsRead(uint(taskID), surveyorID.(uint))

	// 2. 现场解析并返回 GeoJSON
	geoJsonData, err := service.ParseTaskShpToGeoJSON(uint(taskID), surveyorID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "解析成功",
		"data":    geoJsonData,
	})
}
