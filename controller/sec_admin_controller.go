package controller

import (
	"net/http"

	"Field_Survey_Backend/service"
	"Field_Survey_Backend/utils"

	"github.com/gin-gonic/gin"
)

// UploadBasicShpHandler Req 1: 上传全国底图
func UploadBasicShpHandler(c *gin.Context) {
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传包含shp的全国底图ZIP压缩包"})
		return
	}
	defer file.Close()

	if err := service.UploadBasicShp(file, fileHeader); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "全国生态底图上传与解析成功"})
}

// GetBasicShpListHandler Req 2: 获取可用底图列表
func GetBasicShpListHandler(c *gin.Context) {
	paths, err := service.GetBasicShpList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取底图列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": paths})
}

// GetBasicShpGeoJSONHandler Req 3: 解析指定底图为 GeoJSON (由于数据可能非常巨大，前端应搭配 loading 遮罩)
func GetBasicShpGeoJSONHandler(c *gin.Context) {
	shpPath := c.Query("shp_path") // 前端从上一个接口拿到的路径，如 "uploads/basic/eco_zones/eco.shp"
	if shpPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "必须提供 shp_path 参数"})
		return
	}

	// 复用昨天的万能解析器
	geoJsonData := utils.SingleShpToGeoJSON(shpPath)

	c.JSON(http.StatusOK, gin.H{
		"message": "底图加载成功",
		"data":    geoJsonData,
	})
}

// AssignWorkspaceReq Req 4 修改版: 分发工作区并进行后台裁切
type AssignWorkspaceReq struct {
	ProjectID          uint   `json:"project_id" binding:"required"`
	WorkspaceName      string `json:"workspace_name" binding:"required"` // 需要和 "三级名" 一致
	ThirdAdminUsername string `json:"third_admin_username" binding:"required"`
	BasicShpPath       string `json:"basic_shp_path" binding:"required"` // 前端选择的底图路径
	Description        string `json:"description"`
}

func AssignWorkspaceHandler(c *gin.Context) {
	secAdminID, _ := c.Get("userID")

	var req AssignWorkspaceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，请检查输入"})
		return
	}

	workspace, err := service.AssignWorkspace(req.ProjectID, req.WorkspaceName, req.ThirdAdminUsername, req.Description, req.BasicShpPath, secAdminID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "任务工作区已成功从底图中裁切并下发！",
		"data":    workspace,
	})
}
