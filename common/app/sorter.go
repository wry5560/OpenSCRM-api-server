package app

import (
	"openscrm/app/constants"
)

type Sorter struct {
	// SortField 排序字段
	SortField constants.SortField `form:"sort_field" json:"sort_field" gorm:"-"`
	// SortType 排序类型,asc desc
	SortType constants.SortType `form:"sort_type" json:"sort_type" gorm:"-" validate:"omitempty,oneof=asc desc"`
}

func (o *Sorter) SetDefault() *Sorter {
	// 处理空值和无效的 "undefined" 字符串
	if o.SortField == "" || o.SortField == "undefined" {
		o.SortField = constants.SortFieldID
	}
	if o.SortType == "" || o.SortType == "undefined" {
		o.SortType = constants.SortTypeDesc
	}
	return o
}
