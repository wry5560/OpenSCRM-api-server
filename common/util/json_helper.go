package util

import "encoding/json"

// ToJSONBArray 将切片转换为 PostgreSQL JSONB 兼容的 JSON 字符串
// 用于 @> 操作符的参数
func ToJSONBArray[T any](items []T) string {
	bytes, _ := json.Marshal(items)
	return string(bytes)
}

// ToJSONBSingleArray 将单个值包装成 JSON 数组字符串
// 用于检查 JSONB 数组是否包含某个值
func ToJSONBSingleArray[T any](item T) string {
	bytes, _ := json.Marshal([]T{item})
	return string(bytes)
}

// ToJSONBValue 将值转换为 JSONB 兼容的 JSON 字符串
func ToJSONBValue(item any) string {
	bytes, _ := json.Marshal(item)
	return string(bytes)
}
