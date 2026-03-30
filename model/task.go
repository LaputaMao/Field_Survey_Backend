package model

import "time"

// Task 三管批量派发给调查员任务表
type Task struct {
	ID                 uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID        uint       `gorm:"index;not null" json:"workspace_id"`
	CreatorID          uint       `gorm:"index;not null" json:"creator_id"`
	AssigneeID         uint       `gorm:"index;not null" json:"assignee_id"`
	PlannedLineShpUrl  string     `gorm:"size:500" json:"planned_line_shp_url"`  // 路线 SHP 路径
	PlannedPointShpUrl string     `gorm:"size:500" json:"planned_point_shp_url"` // 规划点 SHP 路径
	Status             string     `gorm:"size:20;default:'pending'" json:"status"`
	IsRead             bool       `gorm:"default:false" json:"is_read"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	FinishedAt         *time.Time `gorm:"type:timestamp" json:"finished_at"` // ⭐ 新增：任务完成时间 (指针类型允许为空)
}
