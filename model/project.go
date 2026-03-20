// Package model model/project.go
package model

import "time"

// Project 二管创建的顶级项目表
type Project struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:200;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatorID   uint      `gorm:"index" json:"creator_id"` // 二管的 User ID
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
