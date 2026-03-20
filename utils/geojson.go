package utils

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
