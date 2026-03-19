// main.go
package main

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/router"
)

func main() {
	// 这里应该有初始化数据库的代码 (DB 初始化)
	config.InitDB()

	r := router.SetupRouter()

	// 启动项目，端口9096
	err := r.Run(":9096")
	if err != nil {
		return
	}
}
