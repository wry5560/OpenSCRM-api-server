package controller

import (
	"openscrm/app/constants"
	"openscrm/app/services"
	"openscrm/common/app"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// StaffFrontendMingDaoHandler 侧边栏明道云接口处理器
type StaffFrontendMingDaoHandler struct {
	api *services.MingDaoYunAPI
}

// NewStaffFrontendMingDaoHandler 创建侧边栏明道云处理器实例
func NewStaffFrontendMingDaoHandler() *StaffFrontendMingDaoHandler {
	return &StaffFrontendMingDaoHandler{
		api: services.NewMingDaoYunAPI(),
	}
}

// FieldConfig 字段配置响应
type FieldConfig struct {
	ID       string                     `json:"id"`
	Name     string                     `json:"name"`
	Type     string                     `json:"type"`
	Editable bool                       `json:"editable"`
	Options  []constants.DropdownOption `json:"options,omitempty"`
}

// GetFieldConfigsResponse 字段配置响应
type GetFieldConfigsResponse struct {
	Fields []FieldConfig `json:"fields"`
}

// GetFieldConfigs 获取客户表字段配置
// @tags 侧边栏-明道云
// @Summary 获取客户表字段配置
// @Description 获取侧边栏展示的客户字段配置，包括字段ID、名称、类型、是否可编辑、选项等
// @Produce json
// @Success 200 {object} app.JSONResult{data=GetFieldConfigsResponse} "成功"
// @Router /api/v1/staff-frontend/mingdao/customer/fields [get]
func (h *StaffFrontendMingDaoHandler) GetFieldConfigs(c *gin.Context) {
	handler := app.NewHandler(c)

	fields := make([]FieldConfig, 0, len(constants.MingDaoYunCustomerDisplayFields))
	for _, field := range constants.MingDaoYunCustomerDisplayFields {
		fc := FieldConfig{
			ID:       field.ID,
			Name:     field.Name,
			Type:     field.Type,
			Editable: field.Editable,
		}
		// 添加下拉选项
		if options, ok := constants.MingDaoYunFieldOptions[field.ID]; ok {
			fc.Options = options
		}
		fields = append(fields, fc)
	}

	handler.ResponseItem(GetFieldConfigsResponse{Fields: fields})
}

// MatchCustomerResponse 客户匹配响应
type MatchCustomerResponse struct {
	Found    bool                    `json:"found"`
	Customer *services.CustomerInfo `json:"customer,omitempty"`
}

// MatchCustomer 根据企微外部联系人ID匹配客户
// @tags 侧边栏-明道云
// @Summary 根据企微外部联系人ID匹配客户
// @Description 根据当前聊天的企微外部联系人ID自动匹配明道云客户记录
// @Produce json
// @Param external_user_id query string true "企微外部联系人ID"
// @Success 200 {object} app.JSONResult{data=MatchCustomerResponse} "成功"
// @Failure 400 {object} app.JSONResult{} "请求错误"
// @Router /api/v1/staff-frontend/mingdao/customer/match [get]
func (h *StaffFrontendMingDaoHandler) MatchCustomer(c *gin.Context) {
	handler := app.NewHandler(c)

	externalUserID := c.Query("external_user_id")
	if externalUserID == "" {
		handler.ResponseBadRequestError(errors.New("external_user_id 参数不能为空"))
		return
	}

	customer, err := h.api.GetCustomerByExternalUserID(externalUserID)
	if err != nil {
		handler.ResponseError(errors.Wrap(err, "匹配客户失败"))
		return
	}

	if customer == nil {
		handler.ResponseItem(MatchCustomerResponse{Found: false})
		return
	}

	// 过滤隐藏字段
	filterHiddenFields(customer)

	handler.ResponseItem(MatchCustomerResponse{
		Found:    true,
		Customer: customer,
	})
}

// SearchCustomerRequest 搜索客户请求
type SearchCustomerRequest struct {
	Keyword   string `form:"keyword" binding:"required"`
	PageSize  int    `form:"page_size"`
	PageIndex int    `form:"page_index"`
}

// SearchCustomersResponse 搜索客户响应
type SearchCustomersResponse struct {
	Items []services.CustomerInfo `json:"items"`
	Total int                     `json:"total"`
}

// SearchCustomers 搜索客户
// @tags 侧边栏-明道云
// @Summary 搜索客户
// @Description 根据手机号、客户编号等关键字搜索客户
// @Produce json
// @Param keyword query string true "搜索关键字（手机号/客户编号）"
// @Param page_size query int false "每页数量，默认10"
// @Param page_index query int false "页码，默认1"
// @Success 200 {object} app.JSONResult{data=SearchCustomersResponse} "成功"
// @Failure 400 {object} app.JSONResult{} "请求错误"
// @Router /api/v1/staff-frontend/mingdao/customer/search [get]
func (h *StaffFrontendMingDaoHandler) SearchCustomers(c *gin.Context) {
	handler := app.NewHandler(c)

	var req SearchCustomerRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		handler.ResponseBadRequestError(errors.Wrap(err, "参数错误"))
		return
	}

	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageIndex <= 0 {
		req.PageIndex = 1
	}

	result, err := h.api.SearchCustomers(req.Keyword, req.PageSize, req.PageIndex)
	if err != nil {
		handler.ResponseError(errors.Wrap(err, "搜索客户失败"))
		return
	}

	// 过滤隐藏字段
	for i := range result.Items {
		filterHiddenFields(&result.Items[i])
	}

	handler.ResponseItem(SearchCustomersResponse{
		Items: result.Items,
		Total: result.Total,
	})
}

// GetCustomerResponse 获取客户详情响应
type GetCustomerResponse struct {
	Customer *services.CustomerInfo `json:"customer"`
}

// GetCustomer 获取客户详情
// @tags 侧边栏-明道云
// @Summary 获取客户详情
// @Description 根据记录ID获取客户详细信息
// @Produce json
// @Param row_id path string true "明道云记录ID"
// @Success 200 {object} app.JSONResult{data=GetCustomerResponse} "成功"
// @Failure 400 {object} app.JSONResult{} "请求错误"
// @Failure 404 {object} app.JSONResult{} "客户不存在"
// @Router /api/v1/staff-frontend/mingdao/customer/{row_id} [get]
func (h *StaffFrontendMingDaoHandler) GetCustomer(c *gin.Context) {
	handler := app.NewHandler(c)

	rowID := c.Param("row_id")
	if rowID == "" {
		handler.ResponseBadRequestError(errors.New("row_id 参数不能为空"))
		return
	}

	customer, err := h.api.GetRowByID(rowID)
	if err != nil {
		handler.ResponseError(errors.Wrap(err, "获取客户详情失败"))
		return
	}

	// 过滤隐藏字段
	filterHiddenFields(customer)

	handler.ResponseItem(GetCustomerResponse{Customer: customer})
}

// UpdateCustomerRequest 更新客户请求
type UpdateCustomerRequest struct {
	Fields []services.UpdateRowControl `json:"fields" binding:"required"`
}

// UpdateCustomer 更新客户信息
// @tags 侧边栏-明道云
// @Summary 更新客户信息
// @Description 更新客户的可编辑字段
// @Accept json
// @Produce json
// @Param row_id path string true "明道云记录ID"
// @Param body body UpdateCustomerRequest true "更新字段列表"
// @Success 200 {object} app.JSONResult{} "成功"
// @Failure 400 {object} app.JSONResult{} "请求错误"
// @Router /api/v1/staff-frontend/mingdao/customer/{row_id} [put]
func (h *StaffFrontendMingDaoHandler) UpdateCustomer(c *gin.Context) {
	handler := app.NewHandler(c)

	rowID := c.Param("row_id")
	if rowID == "" {
		handler.ResponseBadRequestError(errors.New("row_id 参数不能为空"))
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.ResponseBadRequestError(errors.Wrap(err, "参数错误"))
		return
	}

	if len(req.Fields) == 0 {
		handler.ResponseBadRequestError(errors.New("fields 不能为空"))
		return
	}

	if err := h.api.UpdateCustomerFields(rowID, req.Fields); err != nil {
		handler.ResponseError(errors.Wrap(err, "更新客户信息失败"))
		return
	}

	handler.ResponseItem(nil)
}

// BindCustomerRequest 绑定客户请求
type BindCustomerRequest struct {
	ExternalUserID string `json:"external_user_id" binding:"required"`
	StaffID        string `json:"staff_id"`
}

// BindCustomer 绑定客户
// @tags 侧边栏-明道云
// @Summary 绑定客户
// @Description 将企微外部联系人ID绑定到明道云客户记录
// @Accept json
// @Produce json
// @Param row_id path string true "明道云记录ID"
// @Param body body BindCustomerRequest true "绑定信息"
// @Success 200 {object} app.JSONResult{} "成功"
// @Failure 400 {object} app.JSONResult{} "请求错误"
// @Router /api/v1/staff-frontend/mingdao/customer/{row_id}/bind [post]
func (h *StaffFrontendMingDaoHandler) BindCustomer(c *gin.Context) {
	handler := app.NewHandler(c)

	rowID := c.Param("row_id")
	if rowID == "" {
		handler.ResponseBadRequestError(errors.New("row_id 参数不能为空"))
		return
	}

	var req BindCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.ResponseBadRequestError(errors.Wrap(err, "参数错误"))
		return
	}

	if err := h.api.BindCustomer(rowID, req.ExternalUserID, req.StaffID); err != nil {
		handler.ResponseError(errors.Wrap(err, "绑定客户失败"))
		return
	}

	handler.ResponseItem(nil)
}

// filterHiddenFields 过滤隐藏字段
func filterHiddenFields(customer *services.CustomerInfo) {
	if customer == nil || customer.Fields == nil {
		return
	}

	for _, hiddenID := range constants.MingDaoYunCustomerHiddenFields {
		delete(customer.Fields, hiddenID)
	}
}
