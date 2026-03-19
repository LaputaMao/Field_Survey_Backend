package model

import (
	"time"
)

// User 对应 users 表
type User struct {
	ID            uint      `Gorm:"primaryKey;autoIncrement" json:"id"`
	Username      string    `Gorm:"uniqueIndex;not null;size:100" json:"username"`
	Email         string    `Gorm:"size:100" json:"email"`
	Password      string    `Gorm:"not null" json:"-"`            // json:"-" 保证密码不会被返回给前端
	Role          string    `Gorm:"not null;size:20" json:"role"` // "user", "sec_admin", "fir_admin"
	CreatedDate   time.Time `Gorm:"autoCreateTime" json:"created_date"`
	LastLoginDate time.Time `json:"last_login_date"`
	LastIP        string    `Gorm:"size:45" json:"last_ip"`
	CreatorID     uint      `Gorm:"index"`
}
