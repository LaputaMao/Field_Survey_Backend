package utils

import (
	"bytes"
	"io/ioutil"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// GbkToUtf8 将 GBK 编码的中文字符串转换为 UTF-8
func GbkToUtf8(s []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, err := ioutil.ReadAll(reader)
	if err != nil {
		return string(s), err // 如果转换失败，返回原字符串（可能是本来就是utf8）
	}
	return string(d), nil
}
