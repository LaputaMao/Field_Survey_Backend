package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"Field_Survey_Backend/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/jonas-p/go-shp"
)

// AssignTask 三管派发任务并上传 SHP ZIP 并解压 tip 弃用
//func AssignTask(workspaceID uint, assigneeUsername, taskName string, thirdAdminID uint, file multipart.File, fileHeader *multipart.FileHeader) (*model.Task, error) {
//	// 1. 权限与人员校验 (同之前一致)
//	var surveyor model.User
//	if err := config.DB.Where("username = ? AND role = ?", assigneeUsername, "user").First(&surveyor).Error; err != nil {
//		return nil, errors.New("未找到对应调查员账号")
//	}
//	if surveyor.CreatorID != thirdAdminID {
//		return nil, errors.New("越权操作：该调查员不属于您管辖")
//	}
//
//	// 2. 将上传的 ZIP 文件先暂存起来
//	tempZipPath := fmt.Sprintf("./uploads/temp_%d_%s", time.Now().Unix(), fileHeader.Filename)
//	os.MkdirAll("./uploads", 0755) // 确保外层目录存在
//	tempZipFile, err := os.Create(tempZipPath)
//	if err != nil {
//		return nil, errors.New("在服务器建立缓存文件失败")
//	}
//	io.Copy(tempZipFile, file)
//	tempZipFile.Close()
//	defer func(name string) {
//		err := os.Remove(name)
//		if err != nil {
//
//		}
//	}(tempZipPath) // ⭐ 无论成功失败，本函数执行完自动删掉 ZIP 压缩包
//
//	// 3. 执行核心逻辑：解压，提取出对应的真实文件夹
//	// 为该任务新建一个专属的隔离文件夹，防止文件名冲突
//	destDir := fmt.Sprintf("./uploads/shps/task_%d_user_%s", time.Now().Unix(), surveyor.Username)
//	shpFilePath, err := utils.UnzipAndFindShp(tempZipPath, destDir)
//	if err != nil {
//		return nil, fmt.Errorf("解析压缩包失败 (请确保包含 .shp 等同名文件): %v", err)
//	}
//
//	// 4. 将解压出来的 .shp 核心文件的绝对/相对地址写入数据库
//	task := model.Task{
//		WorkspaceID:   workspaceID,
//		CreatorID:     thirdAdminID,
//		AssigneeID:    surveyor.ID,
//		Name:          taskName,
//		PlannedShpUrl: shpFilePath, // ⭐ 这里存储的是：./uploads/shps/task_123_user_xxx/路线数据.shp
//		IsRead:        false,
//	}
//
//	if err := config.DB.Create(&task).Error; err != nil {
//		return nil, errors.New("任务派发写入数据库失败")
//	}
//
//	return &task, nil
//}

// TaskDetailResponse 用于包裹给 App 端的完整地图数据
type TaskDetailResponse struct {
	PlannedGeoJSON *utils.FeatureCollection `json:"planned_geojson"`
	// ⭐ 从 []string 修改为包含 JSON 对象的 Map 切片
	ActualLineGeoms []map[string]interface{} `json:"actual_line_geoms"`
	// ⭐ 新增：实际调查打点的精简空间结构
	ActualPointGeoms []map[string]interface{} `json:"actual_point_geoms"`
	TaskStatus       string                   `json:"task_status"`
}

// GetSurveyorTasks 调查员获取自己的任务
func GetSurveyorTasks(surveyorID uint) ([]model.Task, error) {
	var tasks []model.Task
	err := config.DB.Where("assignee_id = ?", surveyorID).Order("created_at desc").Find(&tasks).Error
	return tasks, err
}

// MarkTaskAsRead 标记任务为已读
func MarkTaskAsRead(taskID, surveyorID uint) error {
	return config.DB.Model(&model.Task{}).
		Where("id = ? AND assignee_id = ?", taskID, surveyorID).
		Update("is_read", true).Error
}

// ParseTaskShpToGeoJSON 解析并合并线和点SHP，且【只返回属于该调查员本人】的要素
func ParseTaskShpToGeoJSON(taskID, surveyorID uint) (*utils.FeatureCollection, error) {
	var task model.Task
	if err := config.DB.Where("id = ? AND assignee_id = ?", taskID, surveyorID).First(&task).Error; err != nil {
		return nil, errors.New("任务不存在或无权访问")
	}

	featureCollection := &utils.FeatureCollection{
		Type:     "FeatureCollection",
		Features: []utils.Feature{},
	}

	// 定义一个内部工具函数，专门读取单个shp拼入集合中
	appendFeatures := func(shpUrl string) {
		if shpUrl == "" {
			return
		}
		shape, err := shp.Open(shpUrl)
		if err != nil {
			return
		}
		defer shape.Close()

		fields := shape.Fields()
		for shape.Next() {
			n, p := shape.Shape()
			feature := utils.Feature{
				Type:       "Feature",
				Properties: make(map[string]interface{}),
			}

			switch geom := p.(type) {
			case *shp.PolyLine:
				feature.Geometry.Type = "LineString"
				var coords [][]float64
				for _, pt := range geom.Points {
					coords = append(coords, []float64{pt.X, pt.Y})
				}
				feature.Geometry.Coordinates = coords

			case *shp.Point:
				feature.Geometry.Type = "Point"
				feature.Geometry.Coordinates = []float64{geom.X, geom.Y}
			default:
				continue
			}

			for k, f := range fields {
				fieldName := strings.TrimSpace(f.String())
				strVal := shape.ReadAttribute(n, k)
				feature.Properties[fieldName] = strings.TrimSpace(strVal)
			}
			featureCollection.Features = append(featureCollection.Features, feature)
		}
	}

	// 依次合并线和点的数据，App 端会收到一个大而全的 JSON 供完全渲染！
	appendFeatures(task.PlannedLineShpUrl)
	appendFeatures(task.PlannedPointShpUrl)

	return featureCollection, nil
}

// BulkAssignTasks 批量上传ZIP，解析SHP，依人聚合生成/更新任务
func BulkAssignTasks(workspaceID, thirdAdminID uint, file multipart.File, fileHeader *multipart.FileHeader) error {
	// 1. 暂存 ZIP 文件
	tempZipPath := fmt.Sprintf("./uploads/temp_%d_%s", time.Now().Unix(), fileHeader.Filename)
	os.MkdirAll("./uploads", 0755)
	tempZipFile, err := os.Create(tempZipPath)
	if err != nil {
		return errors.New("服务器建立缓存文件失败")
	}
	io.Copy(tempZipFile, file)
	tempZipFile.Close()
	defer os.Remove(tempZipPath) // 执行后自动删除压缩包

	// 2. 解压并获取所有的 .shp 文件集合
	destDir := fmt.Sprintf("./uploads/shps/workspace_%d_batch_%d", workspaceID, time.Now().Unix())
	shpFiles, err := utils.UnzipAndFindAllShps(tempZipPath, destDir)
	if err != nil || len(shpFiles) == 0 {
		return errors.New("解析压缩包失败，内部未包含 .shp 文件")
	}

	// 3. 遍历每一个 SHP 文件
	for _, shpPath := range shpFiles {
		// 为了确保文件句柄不泄漏，在闭包内执行单个文件的读取逻辑
		func() {
			shape, err := shp.Open(shpPath)
			if err != nil {
				return // 略过损坏的 SHP
			}
			defer shape.Close()

			// 3.1 识别这是 点(Point) 还是 线(Line)
			isLine := shape.GeometryType == shp.POLYLINE || shape.GeometryType == shp.POLYLINEZ
			isPoint := shape.GeometryType == shp.POINT || shape.GeometryType == shp.POINTZ || shape.GeometryType == shp.MULTIPOINT

			if !isLine && !isPoint {
				return // 暂不处理面或其他几何类型
			}

			// 3.2 找到 DBF 中名为 "调查员" 的字段索引
			fields := shape.Fields()
			surveyorFieldIndex := -1
			for k, f := range fields {
				fieldName := strings.TrimSpace(f.String())
				if fieldName == "调查员" {
					surveyorFieldIndex = k
					break
				}
			}

			if surveyorFieldIndex == -1 {
				return // 没有调查员属性，无法分发，略过
			}

			// 3.3 读取第一个要素（第一行）的“调查员”名字即可
			// 因为通常一个 SHP 对应一个人的点集或线集
			var surveyorUsername string
			if shape.Next() {
				n, _ := shape.Shape()
				val := shape.ReadAttribute(n, surveyorFieldIndex)
				// 小鱼提示：这里默认用 UTF-8。如果你在 Windows 用 QGIS/ArcGIS 直接导出遇到乱码，
				// 可以使用 utils.GbkToUtf8([]byte(val)) 进行转码。
				surveyorUsername = strings.TrimSpace(val)
			}

			if surveyorUsername == "" {
				return
			}

			// 3.4 往 users 表查询该调查员的 UserID
			var user model.User
			if err := config.DB.Where("username = ? AND role = ?", surveyorUsername, "user").First(&user).Error; err != nil {
				return // 查无此人，略过
			}
			if user.CreatorID != thirdAdminID {
				return // 防止三管越权跨组分发任务
			}

			// 3.5 核心：聚合寻找 Task 行，无则建，有则改 (Upsert 思想)
			var task model.Task
			config.DB.Where("workspace_id = ? AND assignee_id = ?", workspaceID, user.ID).First(&task)

			// 填充基础信息
			task.WorkspaceID = workspaceID
			task.AssigneeID = user.ID
			task.CreatorID = thirdAdminID
			task.IsRead = false // 有新材料进来了，强制让 APP 端产生红点

			// 根据类型填入对应的 URL
			if isLine {
				task.PlannedLineShpUrl = shpPath
			} else if isPoint {
				task.PlannedPointShpUrl = shpPath
			}

			// 保存回数据库
			if task.ID == 0 {
				config.DB.Create(&task)
			} else {
				config.DB.Save(&task)
			}
		}()
	}

	return nil
}

// GetTaskDetail 获取任务详情
func GetTaskDetail(taskID, surveyorID uint) (*TaskDetailResponse, error) {
	var task model.Task
	if err := config.DB.Where("id = ? AND assignee_id = ?", taskID, surveyorID).First(&task).Error; err != nil {
		return nil, errors.New("任务不存在或无权访问")
	}

	// 1. 获取规划 GeoJSON
	plannedGeoJSON, err := ParseTaskShpToGeoJSON(taskID, surveyorID)
	if err != nil {
		plannedGeoJSON = &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}}
	}

	// 2. 从 actual_route 表中拉取所有历史轨迹
	var actualRoutes []model.ActualRoute
	config.DB.Where("task_id = ? AND user_id = ?", taskID, surveyorID).Find(&actualRoutes)

	// 定义新的切片存储解析后的 JSON 对象
	var actualGeoms []map[string]interface{}
	for _, route := range actualRoutes {
		// 防御性判断，跳过空数据
		if route.ActualLineGeom == "" {
			continue
		}

		var featureObj map[string]interface{}
		// 将文本形态的 JSON 解析为 Go 内存里的 Map 对象
		err := json.Unmarshal([]byte(route.ActualLineGeom), &featureObj)
		if err == nil {
			actualGeoms = append(actualGeoms, featureObj)
		}
	}

	// 3. ⭐ 新增：从 points 表中拉取轻量级的打点地理信息 (忽略 properties)
	type PointRecord struct {
		ID          uint
		PathID      string
		Type        int
		PointSerial string
		GeomJson    string // 接收 PostGIS 返回的 GeoJSON 文本
	}
	var ptRecords []PointRecord

	// 使用 ST_AsGeoJSON 将空间点直接转为标准 JSON 字符串
	config.DB.Raw("SELECT id, path_id, type, point_serial, ST_AsGeoJSON(geom) as geom_json FROM points WHERE task_id = ? AND user_id = ?", taskID, surveyorID).Scan(&ptRecords)

	var actualPtGeoms []map[string]interface{}
	for _, pt := range ptRecords {
		var geometryObj map[string]interface{}
		if err := json.Unmarshal([]byte(pt.GeomJson), &geometryObj); err == nil {
			// 将其封装为标准的 GeoJSON Feature 格式返给 App
			feature := map[string]interface{}{
				"type":     "Feature",
				"geometry": geometryObj,
				"properties": map[string]interface{}{
					"point_id":     pt.ID,
					"path_id":      pt.PathID,
					"type":         pt.Type,
					"point_serial": pt.PointSerial,
				},
			}
			actualPtGeoms = append(actualPtGeoms, feature)
		}
	}

	return &TaskDetailResponse{
		PlannedGeoJSON:   plannedGeoJSON,
		ActualLineGeoms:  actualGeoms,   // 这里装进去的已经是结构化的对象了
		ActualPointGeoms: actualPtGeoms, // ⭐ 将封装好的点位装入返回体
		TaskStatus:       task.Status,
	}, nil
}

// CompleteTask 结束任务
func CompleteTask(taskID, surveyorID uint) error {
	now := time.Now()

	// 使用 map 的方式进行 Updates，可以安全地无视 GORM 零值过滤机制，强行写入 updated_at 和 finished_at
	res := config.DB.Model(&model.Task{}).
		Where("id = ? AND assignee_id = ?", taskID, surveyorID).
		Updates(map[string]interface{}{
			"status":      "completed",
			"finished_at": now,
		})

	if res.Error != nil {
		return errors.New("操作数据库失败")
	}
	if res.RowsAffected == 0 {
		return errors.New("任务不存在或您无权操作该任务")
	}

	return nil
}
