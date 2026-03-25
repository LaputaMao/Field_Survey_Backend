package service

import (
	"database/sql"
	"fmt"
	"time"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
)

// AutoFill 空间查询与规则编号生成
func AutoFill(lon, lat float64, fields []string, routeID string, pointType string) (map[string]interface{}, error) {
	resultMap := make(map[string]interface{})

	// 1. ======= 固定规则生成点号/剖面号 =======
	// (这部分实现了你说的：根据路线号和固定规则生成的逻辑)
	// 比如规则为： 路线号 + 类型简写 + 当前时间戳简写
	typePrefix := "P" // 默认普通点
	if pointType == "剖面点" {
		typePrefix = "S" // Section
	}
	timeSuffix := time.Now().Format("01021504") // 月日时分

	generatedPointID := fmt.Sprintf("%s-%s-%s", routeID, typePrefix, timeSuffix)
	resultMap["自动生成点号"] = generatedPointID

	// 2. ======= 动态空间相交填表 =======

	if len(fields) == 0 {
		return resultMap, nil // 如果只为了请求固定编号，直接返回
	}

	// 提前把前端需要的规则一次性查出来，减少查注册表的耗时
	var registries []model.ReferenceRegistry
	if err := config.DB.Where("data_name IN ?", fields).Find(&registries).Error; err != nil {
		return resultMap, nil
	}

	// 建立 map 加速匹配
	regMap := make(map[string]string)
	for _, r := range registries {
		regMap[r.DataName] = r.QuerySql
	}

	// 遍历前端请求的每一个字段，执行对应的 PostGIS 查询
	//for _, fieldName := range fields {
	//	sqlTemplate, exists := regMap[fieldName]
	//	if !exists || sqlTemplate == "" {
	//		resultMap[fieldName] = "未配置该数据源"
	//		continue
	//	}
	//
	//	var queryResult sql.NullString // 兼容查不到数据时返回 NULL 的情况
	//
	//	// ⭐ 调用 GORM 的 Raw 注入 @lon 和 @lat 进行原生 PG 空间计算
	//	err := config.DB.Raw(sqlTemplate, sql.Named("lon", lon), sql.Named("lat", lat)).Scan(&queryResult).Error
	//
	//	if err != nil {
	//		resultMap[fieldName] = "计算错误"
	//	} else if !queryResult.Valid {
	//		resultMap[fieldName] = "无数据(可能不在范围内)"
	//	} else {
	//		resultMap[fieldName] = queryResult.String
	//	}
	//}
	// 遍历前端请求的每一个字段，执行对应的 PostGIS 查询
	for _, fieldName := range fields {
		sqlTemplate, exists := regMap[fieldName]
		if !exists || sqlTemplate == "" {
			resultMap[fieldName] = "未配置该数据源"
			continue
		}

		// 1. 获取 Rows 迭代器，这样可以处理任意数量的列
		rows, err := config.DB.Raw(sqlTemplate, sql.Named("lon", lon), sql.Named("lat", lat)).Rows()
		if err != nil {
			resultMap[fieldName] = "SQL执行失败"
			continue
		}
		defer rows.Close()

		if rows.Next() {
			// 获取列名
			cols, _ := rows.Columns()

			// 如果只有一列，直接存入字符串（保持你原有的逻辑）
			if len(cols) == 1 {
				var val sql.NullString
				rows.Scan(&val)
				if val.Valid {
					resultMap[fieldName] = val.String
				} else {
					resultMap[fieldName] = "无数据"
				}
			} else {
				// ⭐ 如果有多列，存入一个子 Map
				columns, _ := rows.Columns()
				values := make([]interface{}, len(columns))
				valuePtrs := make([]interface{}, len(columns))
				for i := range columns {
					valuePtrs[i] = &values[i]
				}

				rows.Scan(valuePtrs...)

				subMap := make(map[string]interface{})
				for i, col := range columns {
					var v interface{}
					val := values[i]
					b, ok := val.([]byte)
					if ok {
						v = string(b) // 将数据库驱动返回的 []byte 转为 string
					} else {
						v = val
					}
					subMap[col] = v
				}
				resultMap[fieldName] = subMap
			}
		} else {
			resultMap[fieldName] = "无数据(可能不在范围内)"
		}
	}

	return resultMap, nil
}
