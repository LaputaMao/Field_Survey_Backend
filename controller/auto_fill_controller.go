package controller

import (
	"net/http"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// AutoFillRequest 放在 controller 或 dto 包中
type AutoFillRequest struct {
	RouteID   string   `json:"route_id"`   // App端当前选择的路线号 (用于生成点号)
	PointType string   `json:"point_type"` // 比如 "调查点" 或 "剖面点"
	Longitude float64  `json:"longitude" binding:"required"`
	Latitude  float64  `json:"latitude" binding:"required"`
	Fields    []string `json:"fields"` // 比如 ["所属生态区", "地貌类型", "降水量", "地面高程"]
}

func AutoFillAttrHandler(c *gin.Context) {
	var req AutoFillRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误(要求提供 longitude, latitude 和 fields 数组)"})
		return
	}

	autoFilledData, err := service.AutoFill(req.Longitude, req.Latitude, req.Fields, req.RouteID, req.PointType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "后台计算填表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "自动填表数据生成成功",
		"data":    autoFilledData,
	})
}
