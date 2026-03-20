package utils

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnzipAndFindAllShps 将 zip 解压，并返回目录下找出的所有 .shp 文件的绝对路径列表
func UnzipAndFindAllShps(zipFile, destDir string) ([]string, error) {
	var shpFiles []string

	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	for _, file := range reader.File {
		fpath := filepath.Join(destDir, file.Name)
		// 防御 ZipSlip
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(fpath, file.Mode())
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return nil, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return nil, err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return nil, err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return nil, err
		}

		// ⭐ 如果是以 .shp 结尾的核心文件，推入切片
		if strings.HasSuffix(strings.ToLower(file.Name), ".shp") {
			shpFiles = append(shpFiles, fpath)
		}
	}

	return shpFiles, nil
}
