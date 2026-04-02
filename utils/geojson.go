package utils

import (
	"strings"

	"github.com/jonas-p/go-shp"
)

// FeatureCollection GeoJSON 根结构
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

// Feature 单个要素（一条线或一个点）
type Feature struct {
	Type       string                 `json:"type"`
	Geometry   Geometry               `json:"geometry"`
	Properties map[string]interface{} `json:"properties"` // 这里存放从 DBF 读出的路线编号、名称等属性 ⭐
}

// Geometry 几何体
type Geometry struct {
	Type        string      `json:"type"` // "LineString", "Point" 等
	Coordinates interface{} `json:"coordinates"`
}

// SingleShpToGeoJSON 将单个本地 SHP 文件转为 GeoJSON 标准结构
func SingleShpToGeoJSON(shpPath string) *FeatureCollection {
	fc := &FeatureCollection{Type: "FeatureCollection", Features: []Feature{}}
	if shpPath == "" {
		return fc // 空路径直接返回空集合，防止前端报错
	}

	shape, err := shp.Open(shpPath)
	if err != nil {
		return fc
	}
	defer shape.Close()

	fields := shape.Fields()
	for shape.Next() {
		n, p := shape.Shape()
		feature := Feature{Type: "Feature", Properties: make(map[string]interface{})}

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
		case *shp.Polygon:
			// 面数据解析：PostGIS和标准GeoJSON的 Polygon 要求三维数组 [][][]float64
			// 第一个数组是外沿边界，后续的数组是内部抠除的洞(Holes)
			feature.Geometry.Type = "Polygon"
			var rings [][][]float64
			numParts := len(geom.Parts)

			for i := 0; i < numParts; i++ {
				start := geom.Parts[i]
				var end int32
				if i == numParts-1 {
					end = int32(len(geom.Points))
				} else {
					end = geom.Parts[i+1]
				}

				var ring [][]float64
				for j := start; j < end; j++ {
					pt := geom.Points[j]
					ring = append(ring, []float64{pt.X, pt.Y})
				}
				rings = append(rings, ring)
			}
			feature.Geometry.Coordinates = rings
		default:
			continue
		}

		for k, f := range fields {
			// (如有乱码可见之前的 GbkToUtf8 逻辑，这里略配)
			feature.Properties[strings.TrimSpace(f.String())] = strings.TrimSpace(shape.ReadAttribute(n, k))
		}
		fc.Features = append(fc.Features, feature)
	}
	return fc
}
