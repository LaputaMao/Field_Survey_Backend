package model

import "time"

// ActualRoute 调查员实际行走的轨迹表
type ActualRoute struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID         uint      `gorm:"index;not null" json:"task_id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	PathID         string    `gorm:"size:100;not null" json:"path_id"`           // 对应规划路线的编号(如 R-101)
	ActualLineGeom string    `gorm:"type:text;not null" json:"actual_line_geom"` // ⭐ 直接存储标准 GeoJSON String
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}
