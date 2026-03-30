// main.go
package main

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/router"
	"time"
)

func main() {
	// 这里应该有初始化数据库的代码 (DB 初始化)
	// ⭐ 强制将 Go 程序的本地时区设置为东八区
	var cstZone = time.FixedZone("CST", 8*3600)
	time.Local = cstZone
	config.InitDB()

	r := router.SetupRouter()

	// 启动项目，端口9096
	err := r.Run(":9096")
	if err != nil {
		return
	}
}
