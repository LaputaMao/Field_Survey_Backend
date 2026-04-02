package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonas-p/go-shp"
)

// ExtractFeatureToNewShp 抠图核心：从基础 SHP 提取匹配的要素，生成新的 SHP
// inShpPath: 全国基础 SHP 路径
// outDir: 输出的隔离文件夹
// matchField: 要匹配的字段名 (如 "三级名")
// matchValue: 要匹配的值 (如 workspace_name)
func ExtractFeatureToNewShp(inShpPath, outDir, matchField, matchValue string) (string, error) {
	// 1. 打开原始全国 SHP
	inShape, err := shp.Open(inShpPath)
	if err != nil {
		return "", fmt.Errorf("打开基础SHP失败: %v", err)
	}
	defer inShape.Close()

	// 2. 准备输出目录和输出文件
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}
	outShpName := fmt.Sprintf("%s.shp", matchValue)
	outShpPath := filepath.Join(outDir, outShpName)

	// 3. 创建新的 SHP 写入器 (保持与原本相同的几何类型)
	outShape, err := shp.Create(outShpPath, inShape.GeometryType)
	if err != nil {
		return "", fmt.Errorf("创建新SHP失败: %v", err)
	}
	defer outShape.Close()

	// 4. 完全复制原有的 DBF 属性表结构
	fields := inShape.Fields()
	outShape.SetFields(fields)

	// 5. 寻找匹配的字段索引 ("三级名")
	targetIndex := -1
	for i, f := range fields {
		if strings.TrimSpace(f.String()) == matchField {
			targetIndex = i
			break
		}
	}
	if targetIndex == -1 {
		return "", fmt.Errorf("在属性表中未找到字段: %s", matchField)
	}

	// 6. 遍历原始要素，找到名字匹配的那一项
	found := false
	var newShapeIndex int32 = 0

	for inShape.Next() {
		n, p := inShape.Shape()
		val := inShape.ReadAttribute(n, targetIndex)

		// 由于你提到是 utf-8，我们直接比对即可。如果有 GBK 乱码需先转码
		if strings.TrimSpace(val) == matchValue {
			found = true

			// 写入几何图形 (Geometry)
			outShape.Write(p)

			// 写入该图形关联的所有属性 (DBF)
			for i := range fields {
				attrVal := inShape.ReadAttribute(n, i)
				outShape.WriteAttribute(int(newShapeIndex), i, attrVal)
			}
			newShapeIndex++
		}
	}

	if !found {
		// 如果没找到，清理刚才生成的空文件
		os.Remove(outShpPath)
		os.Remove(strings.Replace(outShpPath, ".shp", ".shx", 1))
		os.Remove(strings.Replace(outShpPath, ".shp", ".dbf", 1))
		return "", fmt.Errorf("基础数据中未匹配到 %s='%s' 的区域", matchField, matchValue)
	}

	// 提取成功！返回这个新 shp 的路径
	return outShpPath, nil
}
