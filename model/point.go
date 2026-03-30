package model

import (
	"time"

	"gorm.io/datatypes"
)

// Point 调查员提交的实地调查点位表
type Point struct {
	ID     uint `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID uint `gorm:"index;not null" json:"task_id"`
	UserID uint `gorm:"index;not null" json:"user_id"`
	// ⭐ 新增：绑定对应的路线号与点类型
	PathID string `gorm:"index;size:100" json:"path_id"`
	Type   int    `gorm:"index;not null;default:0" json:"type"`

	// properties 字段：储存所有动态属性（包括生成的照片URL、相交填表结果、用户手填文字）
	Properties datatypes.JSON `gorm:"type:jsonb" json:"properties"`

	// 注意：GORM 默认不认识 PostGIS 的 geometry 类型。
	// 我们在这里留一个抽象占位，实际写库时用 sql.Expr 注入。
	// 但如果后续需要用 GORM 读出来，我们可以用一个外挂结构体，这里先定义为通用接口或略过绑定。
	// 最简单的防报错写法：不在这里声明具体的 struct 字段给 GORM 自动建表，而是手动在 PG 建好扩展字段。
	// 但是为了全自动，我们可以这样写：
	Geom string `gorm:"type:geometry(Point,4326)" json:"-"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
