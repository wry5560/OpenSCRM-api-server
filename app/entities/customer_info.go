package entities

import (
	"openscrm/app/constants"
)

type UpdateCustomerInfoReq struct {
	ExtStaffID    string                        `json:"ext_staff_id" validate:"required"`
	ExtCustomerID string                        `json:"ext_customer_id" validate:"required"`
	Age           int                           `form:"age" json:"age" validate:"omitempty,gte=0,lt=120"`
	Description   string                        `form:"description" json:"description" validate:"omitempty"`
	Email         string                        `form:"email" json:"email" validate:"omitempty,email"`
	PhoneNumber   string                        `form:"phone_number" json:"phone_number" validate:"omitempty"`
	QQ            string                        `form:"qq" json:"qq" validate:"omitempty"`
	Address       string                        `form:"address" json:"address" validate:"omitempty"`
	Birthday      string                        `form:"birthday" json:"birthday" validate:"omitempty"`
	Weibo         string                        `form:"weibo" json:"weibo" validate:"omitempty"`
	RemarkField   constants.CustomerRemarkField `json:"remark_field" form:"remark_field" validate:"omitempty,dive"`
}

type GetCustomerInfoReq struct {
	// 微信客户ID
	ExtCustomerID string `json:"ext_customer_id" form:"ext_customer_id" validate:"required"`
	// 微信员工ID
	ExtStaffID string `form:"ext_staff_id" json:"ext_staff_id" validate:"required"`
}
