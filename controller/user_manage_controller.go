package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// UploadUsersExcelHandler 上传Excel并导入
func UploadUsersExcelHandler(c *gin.Context) {
	// 从 JWT 获取当前操作者信息
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传Excel文件 (参数名: file)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件打开失败"})
		return
	}
	defer file.Close()

	count, err := service.ImportUsersFromExcel(file, creatorID.(uint), creatorRole.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "导入成功", "imported_count": count})
}

// GetUsersHandler 获取管辖列表 (可通过 ?name=xxx 搜索)
func GetUsersHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	searchName := c.Query("name")

	users, err := service.GetManagedUsers(creatorID.(uint), searchName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

// UpdateUserHandler 修改用户信息
func UpdateUserHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	targetIDStr := c.Param("id") // URL 中的被修改用户的ID
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"` // 如果不填就不会修改密码
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
		return
	}

	err = service.UpdateManagedUser(creatorID.(uint), uint(targetID), req.Username, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DeleteUserHandler 删除用户
func DeleteUserHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	targetIDStr := c.Param("id")
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	err = service.DeleteManagedUser(creatorID.(uint), uint(targetID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
