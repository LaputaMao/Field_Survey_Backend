package controller

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// ExportWorkspaceShpHandler 管理员导出指定工作区的三件套包裹
func ExportWorkspaceShpHandler(c *gin.Context) {
	wsIDStr := c.Param("id")
	wsID, _ := strconv.Atoi(wsIDStr)

	if wsID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 workspace_id"})
		return
	}

	// 调取打包核心
	zipPath, err := service.ExportWorkspaceShp(uint(wsID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打包导出失败: " + err.Error()})
		return
	}

	// 注意：只要这个 Handler 执行完毕退出，defer 的代码就会执行，安全删除物理文件
	defer os.Remove(zipPath)

	// 让前端浏览器识别为附件下载，而不是 JSON
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=workspace_%d_export.zip", wsID))
	c.Header("Content-Type", "application/zip")

	// 直接利用 Gin 极其强悍的本地文件写出方法！
	c.File(zipPath)
}
