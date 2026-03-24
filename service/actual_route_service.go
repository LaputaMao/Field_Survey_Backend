package service

import (
	"encoding/json"
	"errors"
	"time"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
)

// UploadActualRoute 保存调查员上传的真实轨迹
func UploadActualRoute(taskID, userID uint, pathID string, geomJson interface{}, startTime, endTime time.Time) (*model.ActualRoute, error) {
	// 1. 简单的越权/归属校验：确保这个任务真的是分配给该用户的
	var task model.Task
	if err := config.DB.Where("id = ? AND assignee_id = ?", taskID, userID).First(&task).Error; err != nil {
		return nil, errors.New("任务不存在或该任务不属于您")
	}

	// 2. 将前端传来的 Map 或 Struct 格式的 GeoJSON 转化为 String 存储
	geomBytes, err := json.Marshal(geomJson)
	if err != nil {
		return nil, errors.New("轨迹 JSON 编码失败")
	}

	// 3. 构造轨迹模型并入库
	route := model.ActualRoute{
		TaskID:         taskID,
		UserID:         userID,
		PathID:         pathID,
		ActualLineGeom: string(geomBytes),
		StartTime:      startTime,
		EndTime:        endTime,
	}

	if err := config.DB.Create(&route).Error; err != nil {
		return nil, errors.New("轨迹写入数据库失败: " + err.Error())
	}

	// [可选项]：如果该任务是第一次上传轨迹，可以将任务状态变更为 "in_progress" (进行中)
	if task.Status == "pending" {
		config.DB.Model(&task).Update("status", "in_progress")
	}

	return &route, nil
}
