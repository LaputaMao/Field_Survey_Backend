package controller

import (
	"net/http"
	"time"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// UploadRouteReq App 端发来的上传轨迹请求体
type UploadRouteReq struct {
	TaskID    uint        `json:"task_id" binding:"required"`
	PathID    string      `json:"path_id" binding:"required"`
	Geom      interface{} `json:"actual_line_geom" binding:"required"` // 接收任意合法的 JSON 对象
	StartTime time.Time   `json:"start_time" binding:"required"`       // 格式推荐 RFC3339 (如 2023-10-01T15:00:00Z)
	EndTime   time.Time   `json:"end_time" binding:"required"`
}

// UploadActualRouteHandler 接收轨迹上传的路由处理方法
func UploadActualRouteHandler(c *gin.Context) {
	// 从 JWT Token 中获取当前调查员 ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权用户"})
		return
	}

	var req UploadRouteReq
	// ShouldBindJSON 会自动解析时间戳并根据 Struct Tag 验证必填项
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误，请检查日期或字段名：", "detail": err.Error()})
		return
	}

	route, err := service.UploadActualRoute(req.TaskID, userID.(uint), req.PathID, req.Geom, req.StartTime, req.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "轨迹同步成功",
		"data": gin.H{
			"route_id": route.ID, // 返回记录ID，方便后期如果断电重连还能继续更新
		},
	})
}
