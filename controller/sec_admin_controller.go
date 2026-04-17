package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/config"
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

// GetSecAdminWorkspaceGeoJSONHandler 二级管理员获取项目下属工作区GeoJSON
func GetSecAdminWorkspaceGeoJSONHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")
	if creatorRole.(string) != "sec_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅二级管理员可访问"})
		return
	}

	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 project_id 参数"})
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil || projectID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id 参数无效"})
		return
	}

	// 验证项目归属：二级管理员只能访问自己创建的项目
	var count int64
	err = config.DB.Table("projects").
		Where("id = ? AND creator_id = ?", projectID, creatorID.(uint)).
		Count(&count).Error
	if err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该项目"})
		return
	}

	// 硬编码的全国三级生态区SHP路径，可根据需要改为前端传递 shp_path 参数
	shpPath := "./uploads/basic/全国三级项目/全国三级项目.shp"

	geoJSON, err := service.GetWorkspaceGeoJSONByProjectID(uint(projectID), shpPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取工作区GeoJSON成功",
		"data":    geoJSON,
	})
}
