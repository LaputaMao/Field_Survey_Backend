package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/dto"
	"Field_Survey_Backend/model"
	"Field_Survey_Backend/utils"
	"encoding/json"
	"errors"
	"time"
)

// GetWorkspaceSurveyorTree ========= 接口1：获取工作区级联人员列表 =========
func GetWorkspaceSurveyorTree(thirdAdminID uint) ([]dto.WorkspaceListResponse, error) {
	var results []dto.WorkspaceListResponse

	// 1. 查出该三管名下的所有工作区
	var workspaces []model.Workspace
	if err := config.DB.Where("assignee_id = ?", thirdAdminID).Find(&workspaces).Error; err != nil {
		return nil, errors.New("查询工作区失败")
	}

	// 2. 遍历构建树状结构
	for _, ws := range workspaces {
		wsItem := dto.WorkspaceListResponse{
			WorkspaceID:   ws.ID,
			WorkspaceName: ws.Name,
			Surveyors:     []dto.SurveyorItem{},
		}

		// 联表查询：Task 结合 User 获取用户名
		// 利用 GORM 的 Raw 或 Select 快速组装
		type Result struct {
			TaskID   uint
			UserID   uint
			Username string
		}
		var rows []Result

		config.DB.Table("tasks t").
			Select("t.id as task_id, u.id as user_id, u.username").
			Joins("JOIN users u ON t.assignee_id = u.id").
			Where("t.workspace_id = ?", ws.ID).
			Scan(&rows)

		for _, r := range rows {
			wsItem.Surveyors = append(wsItem.Surveyors, dto.SurveyorItem{
				TaskID:   r.TaskID,
				UserID:   r.UserID,
				Username: r.Username,
			})
		}
		results = append(results, wsItem)
	}

	return results, nil
}

// GeTaskComprehensiveDetail ========= 接口2：获取全景四维度巨型 JSON =========
func GeTaskComprehensiveDetail(thirdAdminID, taskID, userID uint) (*dto.TaskDetailResponse, error) {
	var resp dto.TaskDetailResponse
	resp.PointStats = make(map[int]int64)

	// 1. 安全校验并获取基础表数据
	var task model.Task
	if err := config.DB.Where("id = ? AND creator_id = ? AND assignee_id = ?", taskID, thirdAdminID, userID).First(&task).Error; err != nil {
		return nil, errors.New("任务不存在或越权访问")
	}
	var user model.User
	config.DB.Where("id = ?", userID).First(&user)

	// 2. 组装基础信息 BasicInfo
	resp.BasicInfo = dto.TaskBasicInfo{
		Username:      user.Username,
		LastLoginDate: user.LastLoginDate,
		LastIP:        user.LastIP,
		Status:        task.Status,
		IsRead:        task.IsRead,
		CreatedAt:     task.CreatedAt,
	}

	// 3. 按类型统计点位数量
	type StatResult struct {
		Type  int
		Count int64
	}
	var stats []StatResult
	config.DB.Table("points").Select("type, count(*) as count").
		Where("task_id = ? AND user_id = ?", taskID, userID).
		Group("type").Scan(&stats)
	for _, s := range stats {
		resp.PointStats[s.Type] = s.Count
	}

	// 4. 解析规划数据 (SHP to GeoJSON)
	resp.GeoData.PlannedLine = utils.SingleShpToGeoJSON(task.PlannedLineShpUrl)
	resp.GeoData.PlannedPoint = utils.SingleShpToGeoJSON(task.PlannedPointShpUrl)

	// 5. 解析实际轨迹 (PostGIS line to GeoJSON)
	//resp.GeoData.ActualLine = &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}}
	//
	//var actualRoute struct {
	//	GeomJson string
	//	EndTime  time.Time
	//}
	//// ⭐ 魔法所在：ST_AsGeoJSON 直接让 PG 吐出规范的 string
	////	config.DB.Raw("SELECT ST_AsGeoJSON(actual_line_geom) as geom_json, end_time FROM actual_routes WHERE task_id = ?", taskID).Scan(&actualRoute)
	//// 修改 SQL，直接读取字段，不需要转换函数
	////config.DB.Raw("SELECT actual_line_geom as geom_json, end_time FROM actual_routes WHERE task_id = ?", taskID).Scan(&actualRoute)
	//if actualRoute.GeomJson != "" {
	//	// 装填 FinishedAt
	//	resp.BasicInfo.FinishedAt = actualRoute.EndTime
	//
	//	var geom utils.Geometry
	//	err := json.Unmarshal([]byte(actualRoute.GeomJson), &geom)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	feat := utils.Feature{
	//		Type:       "Feature",
	//		Geometry:   geom,
	//		Properties: map[string]interface{}{"description": "调查员实际徒步轨迹"},
	//	}
	//	resp.GeoData.ActualLine.Features = append(resp.GeoData.ActualLine.Features, feat)
	//}
	var actualRoutes []model.ActualRoute
	config.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Find(&actualRoutes)

	var actualGeoms []map[string]interface{}
	var latestEndTime time.Time

	for _, route := range actualRoutes {
		if route.ActualLineGeom == "" {
			continue
		}

		var featureObj map[string]interface{}
		// 你的核心代码：将文本形态的 JSON 解析为 Go 内存里的 Map 对象
		err := json.Unmarshal([]byte(route.ActualLineGeom), &featureObj)
		if err == nil {
			actualGeoms = append(actualGeoms, featureObj)
		}

		// 顺带挑出所有轨迹中最晚的一条结束时间，作为任务最后更新时间
		if route.EndTime.After(latestEndTime) {
			latestEndTime = route.EndTime
		}
	}

	resp.GeoData.ActualLine = actualGeoms
	if !latestEndTime.IsZero() {
		resp.BasicInfo.FinishedAt = latestEndTime
	}

	// 6. 解析实际调查点 (PostGIS points to GeoJSON)
	resp.GeoData.ActualPoints = &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}}

	type PointRecord struct {
		ID       uint // ⭐ 加上 ID
		GeomJson string
		//Properties string // 获取 JSONB 文本
		PathID      string
		Type        int
		PointSerial string
	}
	var ptRecords []PointRecord
	config.DB.Raw("SELECT id, point_serial,ST_AsGeoJSON(geom) as geom_json, path_id, type FROM points WHERE task_id = ?", taskID).Scan(&ptRecords)

	for _, pt := range ptRecords {
		var geom utils.Geometry
		err := json.Unmarshal([]byte(pt.GeomJson), &geom)
		if err != nil {
			return nil, err
		}

		//// 将 JSONB 解码回 map 作为 GeoJSON 的 Properties
		//var props map[string]interface{}
		//err = json.Unmarshal([]byte(pt.Properties), &props)
		//if err != nil {
		//	return nil, err
		//}
		// 2. ⭐ 修改点：不再解析 pt.Properties，直接创建一个新的 map
		props := make(map[string]interface{})
		// 把关键系统标识也塞进属性里给前端显示
		// ⭐ 把 point_id 和 serial 注入到前端可以读取到的 properties 里
		props["point_id"] = pt.ID
		props["point_serial"] = pt.PointSerial
		props["_path_id"] = pt.PathID
		props["_type"] = pt.Type

		feat := utils.Feature{
			Type:       "Feature",
			Geometry:   geom,
			Properties: props,
		}
		resp.GeoData.ActualPoints.Features = append(resp.GeoData.ActualPoints.Features, feat)
	}

	return &resp, nil
}
