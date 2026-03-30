package dto

import (
	"Field_Survey_Backend/utils"
	"time"
)

// SurveyorItem ========= 接口1：工作区及人员列表 DTO =========
type SurveyorItem struct {
	TaskID   uint   `json:"task_id"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

type WorkspaceListResponse struct {
	WorkspaceID   uint           `json:"workspace_id"`
	WorkspaceName string         `json:"name"`
	Surveyors     []SurveyorItem `json:"surveyors"`
}

// TaskBasicInfo ========= 接口2：监控大屏详细数据 DTO =========
type TaskBasicInfo struct {
	Username      string    `json:"username"`
	LastLoginDate time.Time `json:"last_login_date"`
	LastIP        string    `json:"last_ip"`
	Status        string    `json:"status"` // pending, in_progress, completed
	IsRead        bool      `json:"is_read"`
	CreatedAt     time.Time `json:"created_at"`
	FinishedAt    time.Time `json:"finished_at"` // 如果 completed，取 aktual_routes 的 EndTime 或 Task 更新时间
}

type TaskDetailResponse struct {
	BasicInfo  TaskBasicInfo `json:"basic_info"`
	PointStats map[int]int64 `json:"point_stats"` // 统计: {"1": 15, "2": 3} 代表类型1有15个点

	// 四大数据图层
	GeoData struct {
		PlannedLine  *utils.FeatureCollection `json:"planned_line"`  // SHP解析而来
		PlannedPoint *utils.FeatureCollection `json:"planned_point"` // SHP解析而来
		ActualLine   []map[string]interface{} `json:"actual_line"`   // DB的ST_AsGeoJSON而来
		ActualPoints *utils.FeatureCollection `json:"actual_points"` // DB聚合而来
	} `json:"geo_data"`
}
