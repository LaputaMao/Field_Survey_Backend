package service

import (
	"encoding/json"
	"errors"
	"time"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/dto"
	"Field_Survey_Backend/model"
)

// GetPaginatedPoints 多条件分页查询点位精简列表
func GetPaginatedPoints(page, pageSize int, username, dateStr, typeStr string) ([]dto.PointListItem, int64, error) {
	// 联表查询：我们需要 points 所有的字段，并去 users 表抓一个 username 过来
	query := config.DB.Table("points").
		Select("points.id as point_id, points.task_id, points.path_id, points.type, points.point_serial, points.created_at, users.username").
		Joins("left join users on points.user_id = users.id")

	// --- 筛选叠加判断 ---
	// 1. Username 筛选
	if username != "" {
		// 模糊搜索用户名
		query = query.Where("users.username LIKE ?", "%"+username+"%")
	}

	// 2. 日期精确到天筛选 (转化成 大于等于当日0点，小于次日0点)
	if dateStr != "" {
		// 期待前端传入格式为 "2023-10-24"
		startOfDay, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err == nil {
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("points.created_at >= ? AND points.created_at < ?", startOfDay, endOfDay)
		}
	}

	// 3. Type 筛选
	if typeStr != "" {
		query = query.Where("points.type = ?", typeStr)
	}

	// 执行获得总行数 (用于前端分页组件El-Pagination判断总页数)
	var total int64
	query.Count(&total)

	// 分页查询数据
	var items []dto.PointListItem
	offset := (page - 1) * pageSize
	err := query.Order("points.created_at desc").Offset(offset).Limit(pageSize).Scan(&items).Error

	return items, total, err
}

// GetPointPropertiesById 单独调取沉重的 properties 字段
func GetPointPropertiesById(pointID uint) (map[string]interface{}, error) {
	var point model.Point
	// GORM 中运用 Select 只拉取一列，极大提高性能
	if err := config.DB.Select("properties").Where("id = ?", pointID).First(&point).Error; err != nil {
		return nil, errors.New("点位不存在")
	}

	// JSONB 落盘为字节数组，解析回 Map 给前端
	var propsMap map[string]interface{}
	err := json.Unmarshal(point.Properties, &propsMap)
	return propsMap, err
}

// GetSurveyorPointPropertiesById 调查员获取自己提交的表单详细属性
func GetSurveyorPointPropertiesById(pointID, userID uint) (map[string]interface{}, error) {
	var point model.Point

	// ⭐ 加入 user_id = userID 的二次校验，防止横向越权操作
	if err := config.DB.Select("properties").Where("id = ? AND user_id = ?", pointID, userID).First(&point).Error; err != nil {
		return nil, errors.New("点位不存在或无权访问")
	}

	var propsMap map[string]interface{}
	err := json.Unmarshal(point.Properties, &propsMap)
	return propsMap, err
}
