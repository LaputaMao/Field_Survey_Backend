package service

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"

	"github.com/jonas-p/go-shp"
)

func ExportWorkspaceShp(workspaceID uint) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	tempDir := fmt.Sprintf("./uploads/temp_export/ws_%d_%s", workspaceID, timestamp)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建导出临时目录失败: %v", err)
	}
	zipFilePath := fmt.Sprintf("./uploads/temp_export/workspace_%d_survey_data_%s.zip", workspaceID, timestamp)

	// ⭐ 听取你的优化：第一步，先把指定工作区下的所有 Task ID 抽出来
	var taskIDs []uint
	config.DB.Model(&model.Task{}).Where("workspace_id = ?", workspaceID).Pluck("id", &taskIDs)

	if len(taskIDs) == 0 {
		return "", fmt.Errorf("该工作区下没有下发任何任务")
	}

	// ==================== 阶段 A：生成实际点位 SHP ====================
	type PointExportRecord struct {
		ID          uint
		TaskID      uint
		Username    string
		PathID      string
		Type        int
		PointSerial string
		CreatedAt   time.Time
		Properties  string
		Lon         float64 // 倚仗 PostGIS 解码
		Lat         float64 // 倚仗 PostGIS 解码
	}
	var ptRecords []PointExportRecord

	// 使用刚刚抓取的 taskIDs 数组进行 IN 查询
	config.DB.Raw(`
        SELECT p.id, p.task_id, u.username, p.path_id, p.type, p.point_serial, p.created_at, p.properties,
               ST_X(p.geom) as lon, ST_Y(p.geom) as lat
        FROM points p
        JOIN users u ON p.user_id = u.id
        WHERE p.task_id IN ?
    `, taskIDs).Scan(&ptRecords)

	if len(ptRecords) > 0 {
		ptShpPath := filepath.Join(tempDir, "actual_points.shp")
		ptShp, _ := shp.Create(ptShpPath, shp.POINT)
		ptFields := []shp.Field{
			shp.NumberField("TaskID", 8),
			shp.StringField("User", 20),
			shp.StringField("Serial", 30),
			shp.StringField("PathID", 30),
			shp.NumberField("PType", 4),
			shp.StringField("Date", 20),
		}
		ptShp.SetFields(ptFields)

		for _, pt := range ptRecords {
			geomPt := shp.Point{X: pt.Lon, Y: pt.Lat}
			shapeID := int(ptShp.Write(&geomPt))
			ptShp.WriteAttribute(shapeID, 0, pt.TaskID)
			ptShp.WriteAttribute(shapeID, 1, pt.Username)
			ptShp.WriteAttribute(shapeID, 2, pt.PointSerial)
			ptShp.WriteAttribute(shapeID, 3, pt.PathID)
			ptShp.WriteAttribute(shapeID, 4, pt.Type)
			ptShp.WriteAttribute(shapeID, 5, pt.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		ptShp.Close()
	}

	// ==================== 阶段 B：生成实际轨迹(线段) SHP ====================
	type LineExportRecord struct {
		TaskID   uint
		Username string
		PathID   string
		GeomJson string // 由于是 TEXT 字段，这里是充满未知的 GeoJSON
		Date     time.Time
	}
	var lnRecords []LineExportRecord

	// 同样使用 IN 查询
	config.DB.Raw(`
        SELECT a.task_id, u.username, a.path_id, a.actual_line_geom as geom_json, a.created_at as date
        FROM actual_routes a
        JOIN users u ON a.user_id = u.id
        WHERE a.task_id IN ?
    `, taskIDs).Scan(&lnRecords)

	if len(lnRecords) > 0 {
		lnShpPath := filepath.Join(tempDir, "actual_routes.shp")
		lnShp, _ := shp.Create(lnShpPath, shp.POLYLINE)
		lnFields := []shp.Field{
			shp.NumberField("TaskID", 8),
			shp.StringField("User", 20),
			shp.StringField("PathID", 20),
			shp.StringField("Date", 20),
		}
		lnShp.SetFields(lnFields)

		for _, ln := range lnRecords {
			if ln.GeomJson == "" {
				continue
			}

			// ⭐ 修复反序列化逻辑：采用万能 Map 接口动态抽取 Coordinates
			var lineObj map[string]interface{}
			if err := json.Unmarshal([]byte(ln.GeomJson), &lineObj); err != nil {
				continue
			}

			var coordsInterface interface{}
			// 智能判断 App 传上来的是 Feature 结构还是直接的 LineString
			if lineObj["type"] == "Feature" {
				if geometry, hasGeom := lineObj["geometry"].(map[string]interface{}); hasGeom {
					coordsInterface = geometry["coordinates"]
				}
			} else {
				coordsInterface = lineObj["coordinates"]
			}

			var shpPoints []shp.Point
			if coordsList, ok := coordsInterface.([]interface{}); ok {
				// 遍历坐标数组，防崩类型断言
				for _, ptInf := range coordsList {
					if ptArr, isArr := ptInf.([]interface{}); isArr && len(ptArr) >= 2 {
						lon, _ := ptArr[0].(float64)
						lat, _ := ptArr[1].(float64)
						shpPoints = append(shpPoints, shp.Point{X: lon, Y: lat})
					}
				}
			}

			// 折线必须由两点以上构成
			if len(shpPoints) < 2 {
				continue
			}

			polyLine := shp.NewPolyLine([][]shp.Point{shpPoints})
			shapeID := int(lnShp.Write(polyLine))

			lnShp.WriteAttribute(shapeID, 0, ln.TaskID)
			lnShp.WriteAttribute(shapeID, 1, ln.Username)
			lnShp.WriteAttribute(shapeID, 2, ln.PathID)
			lnShp.WriteAttribute(shapeID, 3, ln.Date.Format("2006-01-02 15:04:05"))
		}
		lnShp.Close()
	}

	// ==================== 补丁守卫：修复 go-shp 的命名 Bug ====================
	// 遍历目录，强行把那些没点号的 "xxdbf", "xxshx" 文件加上点号
	files, _ := os.ReadDir(tempDir)
	for _, f := range files {
		name := f.Name()
		if !strings.Contains(name, ".") {
			var newName string
			if strings.HasSuffix(name, "dbf") {
				newName = strings.TrimSuffix(name, "dbf") + ".dbf"
			} else if strings.HasSuffix(name, "shx") {
				newName = strings.TrimSuffix(name, "shx") + ".shx"
			}
			if newName != "" {
				os.Rename(filepath.Join(tempDir, name), filepath.Join(tempDir, newName))
			}
		}
	}

	// ==================== 阶段 C：打包目标文件夹为 ZIP ====================
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(tempDir, path)
		w, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		fileStream, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileStream.Close()
		_, err = io.Copy(w, fileStream)
		return err
	})

	zipWriter.Close()
	os.RemoveAll(tempDir)

	return zipFilePath, nil
}
