package dto

import "time"

// PointListItem 仅包含基础统计数据，剥离了沉重的 properties 和 geom
type PointListItem struct {
	PointID     uint      `json:"point_id"`
	TaskID      uint      `json:"task_id"`
	Username    string    `json:"username"`
	PathID      string    `json:"path_id"`
	Type        int       `json:"type"`
	PointSerial string    `json:"point_serial"`
	CreatedAt   time.Time `json:"created_at"`
}
