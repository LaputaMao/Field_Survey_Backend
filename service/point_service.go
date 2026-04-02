package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UploadSurveyPoint 处理混合上传逻辑
func UploadSurveyPoint(taskID, userID uint, pathID string, pointType int, pointSerial string, lon, lat float64, propertiesJson string, fileMap map[string][]*multipart.FileHeader) (*model.Point, error) {

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
		TaskID:      taskID,
		UserID:      userID,
		PathID:      pathID,
		Type:        pointType,
		PointSerial: pointSerial,
		Properties:  datatypes.JSON(finalJsonBytes),
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

// 递归提取 JSON 中所有可能为上传图片路径的串
func extractAllPhotoPaths(data interface{}, pathsCollection *[]string) {
	switch val := data.(type) {
	case map[string]interface{}:
		// 如果对象有 "path" 字段并且是 /uploads 开头 (适配前端代码 Map 的写法)
		if pathVal, ok := val["path"]; ok {
			if pathStr, isStr := pathVal.(string); isStr && strings.HasPrefix(pathStr, "/uploads") {
				*pathsCollection = append(*pathsCollection, pathStr)
			}
		}
		// 继续深搜所有键下的子节点
		for _, child := range val {
			extractAllPhotoPaths(child, pathsCollection)
		}
	case []interface{}:
		// 遍历数组
		for _, child := range val {
			extractAllPhotoPaths(child, pathsCollection)
		}
	case string:
		// 兼容老版本只存 "/uploads/xxx.jpg" 纯字符的情况
		if strings.HasPrefix(val, "/uploads") {
			*pathsCollection = append(*pathsCollection, val)
		}
	}
}

// 辅助方法：向 JSON 节点中的指定 Key 注入格式化好的对象
func appendPhotoNode(node map[string]interface{}, key string, newPhotoObj map[string]interface{}) {
	if val, ok := node[key]; ok {
		if slice, isSlice := val.([]interface{}); isSlice {
			node[key] = append(slice, newPhotoObj)
		} else {
			// 防御性：如果原先不是数组，转为数组
			node[key] = []interface{}{val, newPhotoObj}
		}
	} else {
		node[key] = []interface{}{newPhotoObj}
	}
}

// UpdateSurveyPoint ========= 新增：更新点位数据 (识别保留照片并覆盖合并) =========
func UpdateSurveyPoint(pointID, userID uint, propertiesJson string, fileMap map[string][]*multipart.FileHeader) (*model.Point, error) {

	// 1. 获取数据库中旧的点位数据，进行越权拦截和旧数据分析
	var existingPoint model.Point
	if err := config.DB.Where("id = ? AND user_id = ?", pointID, userID).First(&existingPoint).Error; err != nil {
		return nil, errors.New("点位不存在或无权修改")
	}

	// 2. 解析新和旧的 JSON 以便打底
	var oldPropsMap, newPropsMap map[string]interface{}
	json.Unmarshal(existingPoint.Properties, &oldPropsMap)
	if err := json.Unmarshal([]byte(propertiesJson), &newPropsMap); err != nil {
		return nil, errors.New("前端提交的新属性 JSON 格式错误")
	}

	// 3. ⭐ 【核心】提取旧图集合和留存图集合，计算并物理删除差集 (废弃图)
	var oldPaths, retainedPaths []string
	extractAllPhotoPaths(oldPropsMap, &oldPaths)
	extractAllPhotoPaths(newPropsMap, &retainedPaths)

	// 利用 Map 快速判断
	retainedMap := make(map[string]bool)
	for _, rp := range retainedPaths {
		retainedMap[rp] = true
	}

	// 异步物理删除废弃的老照片 (不阻塞主业务)
	go func() {
		for _, op := range oldPaths {
			if !retainedMap[op] {
				// 丢弃！注意：需要把开头的 "/" 去掉换回相对系统的真实路径
				cleanPath := "." + op
				os.Remove(cleanPath)
			}
		}
	}()

	// 4. ⭐ 【核心】处理新的文件上传，并把它们正确“塞进”前面整理好的 newPropsMap 中
	uploadDir := fmt.Sprintf("./uploads/points/user_%d/%s", userID, time.Now().Format("20060102"))
	os.MkdirAll(uploadDir, 0755)

	// 这个正则专门用来捕获 App 端发来的特殊结构体长名字
	// 例如: dynamic_assets_植被_0_照片_叶片 -> 主键: dynamic_assets_植被, 索引: 0, 子键: 照片_叶片
	complexKeyRegex := regexp.MustCompile(`^(dynamic_assets_.+)_(\d+)_(.+)$`)

	for formKey, files := range fileMap {
		if len(files) == 0 {
			continue
		}

		for i, fileHeader := range files {
			ext := filepath.Ext(fileHeader.Filename)
			safeFileName := fmt.Sprintf("upd_p%d_%d_%d%s", pointID, time.Now().UnixNano(), i, ext)
			savePath := filepath.Join(uploadDir, safeFileName)

			if err := saveUploadedFile(fileHeader, savePath); err != nil {
				continue
			}

			newUrl := "/" + filepath.ToSlash(savePath)
			// 为了完美适配 App 端的 flutter 返回的结构 Map<String, String>.from(photo)
			newPhotoNode := map[string]interface{}{"path": newUrl}

			// 解析装填逻辑
			matches := complexKeyRegex.FindStringSubmatch(formKey)
			if len(matches) == 4 {
				// === 分类 B: 属于某一个结构体的内嵌图片 ===
				mainKey := matches[1] // dynamic_assets_植被
				idxStr := matches[2]  // 0
				subKey := matches[3]  // 照片_叶片
				idx, _ := strconv.Atoi(idxStr)

				// 安全地导航进入 newPropsMap 并注入，防止空指针
				if mainListVal, ok := newPropsMap[mainKey]; ok {
					if mainList, isList := mainListVal.([]interface{}); isList && idx < len(mainList) {
						if structNode, isMap := mainList[idx].(map[string]interface{}); isMap {
							// 精准空降！把照片放入属于第 0 个元素的结构体里
							appendPhotoNode(structNode, subKey, newPhotoNode)
						}
					}
				}
			} else {
				// === 分类 A: 在外面最浅层的普通图片 ===
				appendPhotoNode(newPropsMap, formKey, newPhotoNode)
			}
		}
	}

	// 5. 将处理得完美无瑕的新 Map 压回 JSON 入库！
	finalJsonBytes, _ := json.Marshal(newPropsMap)
	existingPoint.Properties = datatypes.JSON(finalJsonBytes)

	// 保存更新
	if err := config.DB.Save(&existingPoint).Error; err != nil {
		return nil, fmt.Errorf("覆盖调查数据写入失败: %v", err)
	}

	return &existingPoint, nil
}
