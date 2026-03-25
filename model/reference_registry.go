package model

// ReferenceRegistry 环境变量字典注册表 (开发者手动在数据库中维护)
type ReferenceRegistry struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	DataName string `gorm:"uniqueIndex;size:100"` // 前端传来的字段名，如: "所属生态区", "地面高程"
	DataType string `gorm:"size:20"`              // "vector" (矢量面数据) 或 "raster" (栅格TIFF数据)

	// 用于溯源该数据是从哪个文件导入的，如 "/data/base/2023生态区.shp"
	//DataUrl   string `gorm:"size:500"`

	// 核心：对应的 PostGIS 执行 SQL 模板。
	// 使用 @lon 和 @lat 作为坐标占位符。
	QuerySql string `gorm:"type:text"`
}
