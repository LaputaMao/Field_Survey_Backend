// Package model model/workspace.go
package model

import "time"

// Workspace 二管分发给三管的工作区表
type Workspace struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProjectID   uint      `gorm:"index;not null" json:"project_id"`
	AssigneeID  uint      `gorm:"index;not null" json:"assignee_id"` // 接收任务的三管 User ID
	Name        string    `gorm:"size:200;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	FileUrl     string    `gorm:"size:500;not null" json:"file_url"` // 固定 shp 文件的下载路径
	IsRead      bool      `gorm:"default:false" json:"is_read"`      // ⭐ 用于实现新任务的“未读提醒”
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
