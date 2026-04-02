package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// UploadPointHandler 接收 App 一次性打包发送的点位数据和图片
func UploadPointHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	// 1. 解析 multipart form (最大内存分配：32MB，超过会存入临时文件)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表单解析失败，请确保使用 multipart/form-data 格式"})
		return
	}
	form := c.Request.MultipartForm

	// 2. 提取文本参数
	taskIDStr := c.PostForm("task_id")
	pathID := c.PostForm("path_id")
	typeStr := c.PostForm("type")
	lonStr := c.PostForm("lon")
	latStr := c.PostForm("lat")
	pointSerial := c.PostForm("point_serial")
	propertiesJson := c.PostForm("properties") // App 将填好的表单转为JSON字符串发过来

	taskID, _ := strconv.Atoi(taskIDStr)
	pointType, _ := strconv.Atoi(typeStr)
	lon, _ := strconv.ParseFloat(lonStr, 64)
	lat, _ := strconv.ParseFloat(latStr, 64)

	if taskID == 0 || pathID == "" || typeStr == "" || propertiesJson == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 3. 提取所有的图片文件映射 (Map 结构)
	// 文件 Map 的键是字段名 (如 "habitat_photos", "profile_photos")
	fileMap := form.File

	// 4. 传给 Service 层进行原子处理
	point, err := service.UploadSurveyPoint(uint(taskID), userID.(uint), pathID, pointType, pointSerial, lon, lat, propertiesJson, fileMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "调查点位及相关图片上传成功！",
		"data":    point,
	})
}

// GetNextPointNumberHandler ========= 1. 获取下一个点位编号 API =========
func GetNextPointNumberHandler(c *gin.Context) {
	// 由于这只是简单的查询，我们可以用 GET 请求获取 query 参数
	taskIDStr := c.Query("task_id")
	pathID := c.Query("path_id")
	typeStr := c.Query("type")

	taskID, _ := strconv.Atoi(taskIDStr)
	pointType, _ := strconv.Atoi(typeStr)

	if taskID == 0 || pathID == "" || typeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数缺失: 需要 task_id, path_id, type"})
		return
	}

	nextCode, err := service.GetNextPointNumber(uint(taskID), pathID, pointType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取成功",
		"data": gin.H{
			"next_code": nextCode,
			// 顺便返回一个拼接好的完整编号供 App 偷懒直接用，比如: R1-1-012
			//"full_code": fmt.Sprintf("%s-%d-%s", pathID, pointType, nextCode),
		},
	})
}

// UpdatePointHandler 一键混合更新表单：修改点位数据
func UpdatePointHandler(c *gin.Context) {
	userID, _ := c.Get("userID")

	idStr := c.Param("id")
	pointID, _ := strconv.Atoi(idStr)

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表单解析失败"})
		return
	}

	// App 把带有 “留存旧图标识” 的 JSON 传回来
	propertiesJson := c.PostForm("properties")
	if propertiesJson == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "properties 不得为空"})
		return
	}

	// 获取可能存在的新上传图片 Map (如果没有新图，这里就是空的，也支持)
	fileMap := c.Request.MultipartForm.File

	// 交给大管家 Service 核心去剥离合并
	updatedPoint, err := service.UpdateSurveyPoint(uint(pointID), userID.(uint), propertiesJson, fileMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "修改且同步图片成功！",
		"data":    updatedPoint,
	})
}
