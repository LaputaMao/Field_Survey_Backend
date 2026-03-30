package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UploadSurveyPoint 处理混合上传逻辑
func UploadSurveyPoint(taskID, userID uint, pathID string, pointType int, lon, lat float64, propertiesJson string, fileMap map[string][]*multipart.FileHeader) (*model.Point, error) {

	// 1. 将前端传来的纯文本 JSON 属性解析为 Map，以便我们往里注入照片 URL
	var propsMap map[string]interface{}
	if err := json.Unmarshal([]byte(propertiesJson), &propsMap); err != nil {
		return nil, errors.New("属性表 JSON 格式解析错误")
	}

	// 2. 准备图片存储目录 (按用户和日期隔离)
	dateStr := time.Now().Format("20060102")
	uploadDir := fmt.Sprintf("./uploads/points/user_%d/%s", userID, dateStr)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, errors.New("创建图片存储目录失败")
	}

	// 3. 处理图片并动态绑定到 JSON 属性中
	// fileMap 的键正是 App 端传来的“字段名”，例如 "生境照片" 或 "标本照片"
	for fieldName, files := range fileMap {
		if len(files) == 0 {
			continue
		}

		var savedUrls []string
		for i, fileHeader := range files {
			// 生成防重复的安全文件名 (任务ID_时间戳_字段名_序号)
			ext := filepath.Ext(fileHeader.Filename)
			safeFileName := fmt.Sprintf("t%d_%d_%s_%d%s", taskID, time.Now().UnixNano(), fieldName, i, ext)
			savePath := filepath.Join(uploadDir, safeFileName)

			// 保存文件到本地磁盘
			if err := saveUploadedFile(fileHeader, savePath); err != nil {
				return nil, fmt.Errorf("保存图片 %s 失败: %v", fieldName, err)
			}

			// 仅存相对路径，方便后续 App 拼接 BaseURL 读取
			savedUrls = append(savedUrls, "/"+filepath.ToSlash(savePath))
		}

		// ⭐ 核心解法：将生成的图片URL数组，直接写入 JSONB 对应的键中！
		// 这样数据库里的 JSON 就变成了：{"植被类型":"高山草甸", "生境照片": ["/uploads/...jpg", "..."]}
		propsMap[fieldName] = savedUrls
	}

	// 4. 将注入图片后的 Map 重新转回 JSON 字节以便存入数据库的 JSONB 列
	finalJsonBytes, _ := json.Marshal(propsMap)

	// 5. 校验任务是否归属该调查员
	var task model.Task
	if err := config.DB.Where("id = ? AND assignee_id = ?", taskID, userID).First(&task).Error; err != nil {
		return nil, errors.New("任务校验失败，您无权向该任务上传点位")
	}

	// 6. 构造 Point 对象入库 (注入 PostGIS 空间 SQL)
	point := model.Point{
		TaskID:     taskID,
		UserID:     userID,
		PathID:     pathID,
		Type:       pointType,
		Properties: datatypes.JSON(finalJsonBytes),
		// Geom 我们不在这里直接赋字符串，而是跳过它在下方用 Update Column 处理
	}

	// 开启事务保证原子性
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 6.1 插入基础数据 (ID, TaskID, UserID, Properties)
		if err := tx.Omit("Geom").Create(&point).Error; err != nil {
			return err
		}

		// 6.2 强行注入 PostGIS 的 ST_MakePoint 原生 SQL 补全 Geom 字段
		geomSql := gorm.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", lon, lat)
		if err := tx.Model(&point).Update("geom", geomSql).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("调查数据写入数据库失败: %v", err)
	}

	return &point, nil
}

// 内部工具函数：将 multipart 文件写入磁盘
func saveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// GetNextPointNumber ========= 1. 新增：获取下一个点位编号 =========
func GetNextPointNumber(taskID uint, pathID string, pointType int) (string, error) {
	var count int64

	// 直接利用 GORM 的 Count 聚合函数去查该任务、该路线、该类型的点有几个
	err := config.DB.Model(&model.Point{}).
		Where("task_id = ? AND path_id = ? AND type = ?", taskID, pathID, pointType).
		Count(&count).Error

	if err != nil {
		return "", errors.New("查询点位数量失败")
	}

	// 核心格式化：count + 1，并自动补齐 3 位前导零 (如 0 -> "001", 11 -> "012")
	nextNumber := fmt.Sprintf("%03d", count+1)
	return nextNumber, nil
}
