package config

import (
	"Field_Survey_Backend/model"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB 全局 DB 变量
var DB *gorm.DB

func InitDB() {
	// 请将这里的参数替换为你实际的 PostgreSQL 数据库信息
	// 例如：user=postgres password=你的密码 dbname=你的数据库名
	// tip 测试url
	dsn := "host=localhost user=postgres password=123456 dbname=field_survey_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	// tip 部署url
	//dsn := "host=127.0.0.1 user=postgres password=your_strong_password dbname=field_survey_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	err = db.AutoMigrate(&model.User{}, &model.Project{}, &model.Workspace{}, &model.Task{}, &model.ActualRoute{}, &model.ReferenceRegistry{}, &model.Point{})
	if err != nil {
		fmt.Printf("自动建表失败: %v", err)
	}

	DB = db
}
