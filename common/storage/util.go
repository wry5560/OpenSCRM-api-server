package storage

import (
	"github.com/pkg/errors"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
)

func GetContentType(filePath string) (contentType string, err error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		err = errors.New("file ext is required")
		return
	}

	contentType = mime.TypeByExtension(ext)
	if contentType == "" {
		err = errors.New("invalid file ext")
		return
	}

	return
}

func IsValidObjectKey(objectKey string) bool {
	// 禁止路径遍历和危险字符，但允许中文等 Unicode 字符
	if strings.Contains(objectKey, "..") || strings.Contains(objectKey, "\\") {
		return false
	}
	// 确保有文件扩展名
	return regexp.MustCompile(`\.[a-zA-Z0-9]+$`).MatchString(objectKey)
}
