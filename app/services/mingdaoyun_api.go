package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"openscrm/app/constants"
	"openscrm/common/log"
	"openscrm/conf"
	"time"

	"github.com/pkg/errors"
)

// MingDaoYunAPI 明道云 API 封装（V3 版本）
type MingDaoYunAPI struct {
	httpClient *http.Client
}

// NewMingDaoYunAPI 创建明道云 API 客户端
func NewMingDaoYunAPI() *MingDaoYunAPI {
	return &MingDaoYunAPI{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MingDaoYunResponse 明道云通用响应结构
type MingDaoYunResponse struct {
	Success   bool        `json:"success"`
	ErrorCode int         `json:"error_code"`
	ErrorMsg  string      `json:"error_msg"`
	Data      interface{} `json:"data"`
}

// ========== V3 API 请求/响应结构 ==========

// V3FilterCondition V3 版本过滤条件
type V3FilterCondition struct {
	Type     string              `json:"type"`               // "group" 或 "condition"
	Logic    string              `json:"logic,omitempty"`    // "AND" 或 "OR"（type=group 时使用）
	Children []V3FilterCondition `json:"children,omitempty"` // 子条件（type=group 时使用）
	Field    string              `json:"field,omitempty"`    // 字段ID（type=condition 时使用）
	Operator string              `json:"operator,omitempty"` // 操作符（type=condition 时使用）
	Value    []string            `json:"value,omitempty"`    // 值（type=condition 时使用）
}

// V3GetRowsListRequest V3 获取行记录列表请求
type V3GetRowsListRequest struct {
	PageSize            int               `json:"pageSize"`
	PageIndex           int               `json:"pageIndex"`
	ViewID              string            `json:"viewId,omitempty"`
	Fields              []string          `json:"fields,omitempty"`
	Filter              *V3FilterCondition `json:"filter,omitempty"`
	Sorts               []V3Sort          `json:"sorts,omitempty"`
	Search              string            `json:"search,omitempty"`
	TableView           bool              `json:"tableView,omitempty"`
	UseFieldIdAsKey     bool              `json:"useFieldIdAsKey,omitempty"`
	IncludeTotalCount   bool              `json:"includeTotalCount,omitempty"`
	IncludeSystemFields bool              `json:"includeSystemFields,omitempty"`
}

// V3Sort V3 排序字段
type V3Sort struct {
	Field string `json:"field"`
	IsAsc bool   `json:"isAsc"`
}

// V3FieldUpdate V3 字段更新结构
type V3FieldUpdate struct {
	ID    string      `json:"id"`
	Value interface{} `json:"value"`
	Type  string      `json:"type,omitempty"` // SingleSelect/MultipleSelect: 1=不增量，2=允许增加
}

// V3CreateRowRequest V3 创建行记录请求
type V3CreateRowRequest struct {
	Fields          []V3FieldUpdate `json:"fields"`
	TriggerWorkflow bool            `json:"triggerWorkflow"`
}

// V3UpdateRowRequest V3 更新行记录请求
type V3UpdateRowRequest struct {
	Fields          []V3FieldUpdate `json:"fields"`
	TriggerWorkflow bool            `json:"triggerWorkflow"`
}

// V3DeleteRowRequest V3 删除行记录请求
type V3DeleteRowRequest struct {
	TriggerWorkflow bool `json:"triggerWorkflow"`
	Permanent       bool `json:"permanent,omitempty"`
}

// ========== 兼容层结构（保持原有接口不变） ==========

// FilterCondition 过滤条件（兼容 V2 格式，内部转换为 V3）
type FilterCondition struct {
	ControlID  string   `json:"controlId"`
	DataType   int      `json:"dataType"`
	SpliceType int      `json:"spliceType"`
	FilterType int      `json:"filterType"`
	Value      string   `json:"value,omitempty"`
	Values     []string `json:"values,omitempty"`
}

// UpdateRowControl 更新行的字段控制（兼容 V2 格式）
type UpdateRowControl struct {
	ControlID string `json:"controlId"`
	Value     string `json:"value"`
}

// MingDaoCustomerInfo 明道云客户信息结构（用于返回给前端）
type MingDaoCustomerInfo struct {
	RowID  string                 `json:"row_id"`
	Fields map[string]interface{} `json:"fields"`
}

// MingDaoCustomerSearchResult 明道云客户搜索结果
type MingDaoCustomerSearchResult struct {
	Items []MingDaoCustomerInfo `json:"items"`
	Total int                   `json:"total"`
}

// ========== 辅助方法 ==========

// doV3Request 执行 V3 API 请求
func (api *MingDaoYunAPI) doV3Request(method, path string, body interface{}) ([]byte, error) {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return nil, errors.New("明道云配置不完整")
	}

	url := fmt.Sprintf("%s%s", cfg.APIBase, path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "序列化请求体失败")
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}

	// V3 API 使用 Header 传递认证信息
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HAP-Appkey", cfg.AppKey)
	req.Header.Set("HAP-Sign", cfg.Sign)

	log.Sugar.Debugw("调用明道云 V3 API",
		"method", method,
		"url", url,
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取响应失败")
	}

	return respBody, nil
}

// convertV2FilterToV3 将 V2 过滤条件转换为 V3 格式
func convertV2FilterToV3(filters []FilterCondition) *V3FilterCondition {
	if len(filters) == 0 {
		return nil
	}

	// V2 filterType 到 V3 operator 的映射
	operatorMap := map[int]string{
		1:  "eq",           // 等于
		2:  "ne",           // 不等于
		3:  "contains",     // 包含
		4:  "notcontains",  // 不包含
		5:  "startswith",   // 开头是
		6:  "endswith",     // 结尾是
		7:  "isempty",      // 为空
		8:  "isnotempty",   // 不为空
		11: "between",      // 在范围内
		12: "notbetween",   // 不在范围内
		13: "gt",           // 大于
		14: "gte",          // 大于等于
		15: "lt",           // 小于
		16: "lte",          // 小于等于
		24: "in",           // 属于
		25: "notin",        // 不属于
	}

	children := make([]V3FilterCondition, 0, len(filters))
	for _, f := range filters {
		operator := operatorMap[f.FilterType]
		if operator == "" {
			operator = "eq" // 默认等于
		}

		condition := V3FilterCondition{
			Type:     "condition",
			Field:    f.ControlID,
			Operator: operator,
		}

		// 处理值
		if f.Value != "" {
			condition.Value = []string{f.Value}
		} else if len(f.Values) > 0 {
			condition.Value = f.Values
		}

		children = append(children, condition)
	}

	// 如果只有一个条件，直接返回
	if len(children) == 1 {
		return &V3FilterCondition{
			Type:     "group",
			Logic:    "AND",
			Children: children,
		}
	}

	// 多个条件时，根据 spliceType 确定逻辑关系
	logic := "AND"
	if len(filters) > 0 && filters[0].SpliceType == 2 {
		logic = "OR"
	}

	return &V3FilterCondition{
		Type:     "group",
		Logic:    logic,
		Children: children,
	}
}

// ========== 工作表结构相关 ==========

// ViewField 视图字段结构（用于前端动态渲染）
type ViewField struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Alias     string        `json:"alias,omitempty"`
	Type      string        `json:"type"`
	SubType   int           `json:"subType,omitempty"`
	Options   []FieldOption `json:"options,omitempty"`
	Required  bool          `json:"required"`
	Editable  bool          `json:"editable"`
	IsTitle   bool          `json:"isTitle,omitempty"`
	IsHidden  bool          `json:"isHidden,omitempty"`
	IsReadOnly bool         `json:"isReadOnly,omitempty"`
	Unit      string        `json:"unit,omitempty"`
	Precision int           `json:"precision,omitempty"`
}

// FieldOption 字段选项
type FieldOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index int    `json:"index,omitempty"`
}

// WorksheetInfoResponse 工作表结构响应
type WorksheetInfoResponse struct {
	WorksheetID string     `json:"worksheetId"`
	Name        string     `json:"name"`
	Alias       string     `json:"alias"`
	Views       []ViewInfo `json:"views"`
	Fields      []FieldInfo `json:"fields"`
}

// ViewInfo 视图信息
type ViewInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

// FieldInfo V3 字段信息
type FieldInfo struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Alias      string        `json:"alias"`
	Type       string        `json:"type"`
	Required   bool          `json:"required"`
	IsHidden   bool          `json:"isHidden"`
	IsReadOnly bool          `json:"isReadOnly"`
	IsTitle    bool          `json:"isTitle"`
	SubType    int           `json:"subType"`
	Precision  int           `json:"precision"`
	Options    []FieldOption `json:"options"`
	DataSource string        `json:"dataSource"`
}

// GetWorksheetInfo 获取工作表结构（V3 API）
func (api *MingDaoYunAPI) GetWorksheetInfo(worksheetId string) (*WorksheetInfoResponse, error) {
	path := fmt.Sprintf("/v3/app/worksheets/%s", worksheetId)

	respBody, err := api.doV3Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var mdyResp struct {
		Success   bool   `json:"success"`
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			WorksheetID string `json:"worksheetId"`
			Name        string `json:"name"`
			Alias       string `json:"alias"`
			Views       []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"views"`
			Fields []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Alias      string `json:"alias"`
				Type       string `json:"type"`
				Required   bool   `json:"required"`
				IsHidden   bool   `json:"isHidden"`
				IsReadOnly bool   `json:"isReadOnly"`
				IsTitle    bool   `json:"isTitle"`
				SubType    int    `json:"subType"`
				Precision  int    `json:"precision"`
				Options    []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
					Index int    `json:"index"`
				} `json:"options"`
				DataSource string `json:"dataSource"`
			} `json:"fields"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(respBody), "err", err)
		return nil, errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
		)
		return nil, fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	// 转换为内部结构
	result := &WorksheetInfoResponse{
		WorksheetID: mdyResp.Data.WorksheetID,
		Name:        mdyResp.Data.Name,
		Alias:       mdyResp.Data.Alias,
	}

	for _, v := range mdyResp.Data.Views {
		result.Views = append(result.Views, ViewInfo{
			ID:   v.ID,
			Name: v.Name,
			Type: v.Type,
		})
	}

	for _, f := range mdyResp.Data.Fields {
		field := FieldInfo{
			ID:         f.ID,
			Name:       f.Name,
			Alias:      f.Alias,
			Type:       f.Type,
			Required:   f.Required,
			IsHidden:   f.IsHidden,
			IsReadOnly: f.IsReadOnly,
			IsTitle:    f.IsTitle,
			SubType:    f.SubType,
			Precision:  f.Precision,
			DataSource: f.DataSource,
		}
		for _, opt := range f.Options {
			field.Options = append(field.Options, FieldOption{
				Key:   opt.Key,
				Value: opt.Value,
				Index: opt.Index,
			})
		}
		result.Fields = append(result.Fields, field)
	}

	return result, nil
}

// GetViewFields 获取指定视图的字段列表（用于前端动态渲染）
func (api *MingDaoYunAPI) GetViewFields(worksheetId, viewId string) ([]ViewField, error) {
	// 获取工作表结构
	wsInfo, err := api.GetWorksheetInfo(worksheetId)
	if err != nil {
		return nil, errors.Wrap(err, "获取工作表结构失败")
	}

	// V3 API 返回的字段已包含类型字符串，直接过滤非隐藏字段
	// 注意：V3 API 的视图信息不包含字段列表，需要根据字段的 isHidden 属性过滤
	var fields []ViewField
	for _, f := range wsInfo.Fields {
		// 跳过隐藏字段
		if f.IsHidden {
			continue
		}

		field := ViewField{
			ID:         f.ID,
			Name:       f.Name,
			Alias:      f.Alias,
			Type:       f.Type,
			SubType:    f.SubType,
			Required:   f.Required,
			IsHidden:   f.IsHidden,
			IsReadOnly: f.IsReadOnly,
			IsTitle:    f.IsTitle,
			Precision:  f.Precision,
			Editable:   isV3FieldEditable(f.Type, f.IsReadOnly),
		}

		// 复制选项
		for _, opt := range f.Options {
			field.Options = append(field.Options, FieldOption{
				Key:   opt.Key,
				Value: opt.Value,
				Index: opt.Index,
			})
		}

		fields = append(fields, field)
	}

	log.Sugar.Infow("获取视图字段完成",
		"worksheetId", worksheetId,
		"viewId", viewId,
		"fieldsCount", len(fields),
	)

	return fields, nil
}

// isV3FieldEditable 判断 V3 字段是否可编辑
func isV3FieldEditable(fieldType string, isReadOnly bool) bool {
	if isReadOnly {
		return false
	}

	// 不可编辑的类型
	nonEditableTypes := map[string]bool{
		"AutoNumber":   true, // 自动编号
		"Formula":      true, // 公式
		"Rollup":       true, // 汇总
		"Lookup":       true, // 查找引用
		"DateFormula":  true, // 日期公式
		"QueryRecord":  true, // 查询记录
		"Approval":     true, // 审批流程
	}

	// 布局类型
	layoutTypes := map[string]bool{
		"Divider":  true, // 分段
		"Section":  true, // 分组/标签页
		"Embed":    true, // 嵌入
	}

	return !nonEditableTypes[fieldType] && !layoutTypes[fieldType]
}

// ========== 行记录操作 ==========

// GetFilterRows 查询记录列表（V3 API）
func (api *MingDaoYunAPI) GetFilterRows(filters []FilterCondition, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	return api.GetFilterRowsByWorksheet(constants.MingDaoYunCustomerWorksheetAlias, filters, pageSize, pageIndex)
}

// GetFilterRowsByWorksheet 查询记录列表（通用方法，V3 API）
func (api *MingDaoYunAPI) GetFilterRowsByWorksheet(worksheetId string, filters []FilterCondition, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	path := fmt.Sprintf("/v3/app/worksheets/%s/rows/list", worksheetId)

	reqBody := V3GetRowsListRequest{
		PageSize:          pageSize,
		PageIndex:         pageIndex,
		Filter:            convertV2FilterToV3(filters),
		IncludeTotalCount: true,
		UseFieldIdAsKey:   false, // 使用别名作为 key
	}

	respBody, err := api.doV3Request("POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var mdyResp struct {
		Success   bool   `json:"success"`
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			Rows  []map[string]interface{} `json:"rows"`
			Total int                      `json:"total"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(respBody), "err", err)
		return nil, errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
		)
		return nil, fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	result := &MingDaoCustomerSearchResult{
		Total: mdyResp.Data.Total,
		Items: make([]MingDaoCustomerInfo, 0, len(mdyResp.Data.Rows)),
	}

	for _, row := range mdyResp.Data.Rows {
		// V3 API 返回的行 ID 字段名为 "id"
		rowID, _ := row["id"].(string)
		if rowID == "" {
			rowID, _ = row["rowid"].(string) // 兼容旧格式
		}
		result.Items = append(result.Items, MingDaoCustomerInfo{
			RowID:  rowID,
			Fields: row,
		})
	}

	return result, nil
}

// GetFilterRowsWithKeywords 使用关键字搜索记录（V3 API）
func (api *MingDaoYunAPI) GetFilterRowsWithKeywords(keywords string, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	path := fmt.Sprintf("/v3/app/worksheets/%s/rows/list", constants.MingDaoYunCustomerWorksheetAlias)

	reqBody := V3GetRowsListRequest{
		PageSize:          pageSize,
		PageIndex:         pageIndex,
		Search:            keywords,
		IncludeTotalCount: true,
		UseFieldIdAsKey:   false,
	}

	respBody, err := api.doV3Request("POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var mdyResp struct {
		Success   bool   `json:"success"`
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			Rows  []map[string]interface{} `json:"rows"`
			Total int                      `json:"total"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(respBody), "err", err)
		return nil, errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
		)
		return nil, fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	result := &MingDaoCustomerSearchResult{
		Total: mdyResp.Data.Total,
	}
	for _, row := range mdyResp.Data.Rows {
		rowID, _ := row["id"].(string)
		if rowID == "" {
			rowID, _ = row["rowid"].(string)
		}
		result.Items = append(result.Items, MingDaoCustomerInfo{
			RowID:  rowID,
			Fields: row,
		})
	}

	return result, nil
}

// GetRowByID 根据记录ID获取详情（V3 API）
func (api *MingDaoYunAPI) GetRowByID(rowId string) (*MingDaoCustomerInfo, error) {
	path := fmt.Sprintf("/v3/app/worksheets/%s/rows/%s", constants.MingDaoYunCustomerWorksheetAlias, rowId)

	respBody, err := api.doV3Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var mdyResp struct {
		Success   bool                   `json:"success"`
		ErrorCode int                    `json:"error_code"`
		ErrorMsg  string                 `json:"error_msg"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(respBody), "err", err)
		return nil, errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
			"rowId", rowId,
		)
		return nil, fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	return &MingDaoCustomerInfo{
		RowID:  rowId,
		Fields: mdyResp.Data,
	}, nil
}

// GetCustomerByExternalUserID 根据企微外部联系人ID获取客户
func (api *MingDaoYunAPI) GetCustomerByExternalUserID(externalUserID string) (*MingDaoCustomerInfo, error) {
	if externalUserID == "" {
		return nil, errors.New("externalUserID 不能为空")
	}

	filters := []FilterCondition{
		{
			ControlID:  constants.MingDaoYunCustomerFields["wecomExternalUserid"],
			DataType:   2,
			SpliceType: 1,
			FilterType: 1, // 等于
			Value:      externalUserID,
		},
	}

	result, err := api.GetFilterRows(filters, 1, 1)
	if err != nil {
		return nil, err
	}

	if result.Total == 0 || len(result.Items) == 0 {
		return nil, nil
	}

	return &result.Items[0], nil
}

// SearchCustomers 搜索客户
func (api *MingDaoYunAPI) SearchCustomers(keyword string, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	if keyword == "" {
		return nil, errors.New("搜索关键字不能为空")
	}
	return api.GetFilterRowsWithKeywords(keyword, pageSize, pageIndex)
}

// ========== 行记录更新操作 ==========

// UpdateRow 更新明道云记录（V3 API）
func (api *MingDaoYunAPI) UpdateRow(rowId string, fields map[string]string) error {
	// 构建 V3 字段数组
	v3Fields := make([]V3FieldUpdate, 0, len(fields))
	for fieldName, fieldValue := range fields {
		fieldID, ok := constants.MingDaoYunCustomerFields[fieldName]
		if !ok {
			log.Sugar.Warnw("未知的明道云字段", "fieldName", fieldName)
			continue
		}
		v3Fields = append(v3Fields, V3FieldUpdate{
			ID:    fieldID,
			Value: fieldValue,
		})
	}

	if len(v3Fields) == 0 {
		return errors.New("没有有效的字段需要更新")
	}

	return api.editRowV3(constants.MingDaoYunCustomerWorksheetAlias, rowId, v3Fields)
}

// editRowV3 V3 API 更新行记录
func (api *MingDaoYunAPI) editRowV3(worksheetId, rowId string, fields []V3FieldUpdate) error {
	path := fmt.Sprintf("/v3/app/worksheets/%s/rows/%s", worksheetId, rowId)

	reqBody := V3UpdateRowRequest{
		Fields:          fields,
		TriggerWorkflow: true,
	}

	respBody, err := api.doV3Request("PATCH", path, reqBody)
	if err != nil {
		return err
	}

	var mdyResp MingDaoYunResponse
	if err := json.Unmarshal(respBody, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(respBody), "err", err)
		return errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
			"rowId", rowId,
		)
		return fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	log.Sugar.Infow("明道云记录更新成功", "worksheetId", worksheetId, "rowId", rowId)
	return nil
}

// UpdateCustomerFields 更新客户指定字段（V3 API）
func (api *MingDaoYunAPI) UpdateCustomerFields(rowId string, updates []UpdateRowControl) error {
	if len(updates) == 0 {
		return errors.New("没有有效的字段需要更新")
	}

	// 动态获取视图中的可编辑字段进行验证
	viewFields, err := api.GetViewFields(
		constants.MingDaoYunCustomerWorksheetAlias,
		constants.MingDaoYunCustomerSidebarViewID,
	)
	if err != nil {
		log.Sugar.Warnw("获取视图字段失败，跳过字段可编辑性验证", "err", err)
		viewFields = nil
	}

	var v3Fields []V3FieldUpdate
	if viewFields != nil {
		// 构建可编辑字段映射
		editableFields := make(map[string]bool)
		for _, field := range viewFields {
			if field.Editable {
				editableFields[field.ID] = true
				if field.Alias != "" {
					editableFields[field.Alias] = true
				}
			}
		}

		// 验证并转换更新字段
		for _, update := range updates {
			if editableFields[update.ControlID] {
				v3Fields = append(v3Fields, V3FieldUpdate{
					ID:    update.ControlID,
					Value: update.Value,
				})
			} else {
				log.Sugar.Warnw("尝试更新不可编辑的字段", "controlId", update.ControlID)
			}
		}

		if len(v3Fields) == 0 {
			return errors.New("没有可编辑的字段")
		}
	} else {
		// 没有视图字段信息时，直接使用所有更新
		for _, update := range updates {
			v3Fields = append(v3Fields, V3FieldUpdate{
				ID:    update.ControlID,
				Value: update.Value,
			})
		}
	}

	return api.editRowV3(constants.MingDaoYunCustomerWorksheetAlias, rowId, v3Fields)
}

// EditRowByWorksheet 更新记录（通用方法，V3 API）
func (api *MingDaoYunAPI) EditRowByWorksheet(worksheetId, rowId string, controls []UpdateRowControl) error {
	v3Fields := make([]V3FieldUpdate, 0, len(controls))
	for _, ctrl := range controls {
		v3Fields = append(v3Fields, V3FieldUpdate{
			ID:    ctrl.ControlID,
			Value: ctrl.Value,
		})
	}
	return api.editRowV3(worksheetId, rowId, v3Fields)
}

// ========== 行记录创建/删除操作 ==========

// CreateRow 创建记录（V3 API）
func (api *MingDaoYunAPI) CreateRow(worksheetId string, controls []UpdateRowControl) (string, error) {
	path := fmt.Sprintf("/v3/app/worksheets/%s/rows", worksheetId)

	v3Fields := make([]V3FieldUpdate, 0, len(controls))
	for _, ctrl := range controls {
		v3Fields = append(v3Fields, V3FieldUpdate{
			ID:    ctrl.ControlID,
			Value: ctrl.Value,
		})
	}

	reqBody := V3CreateRowRequest{
		Fields:          v3Fields,
		TriggerWorkflow: true,
	}

	respBody, err := api.doV3Request("POST", path, reqBody)
	if err != nil {
		return "", err
	}

	var mdyResp struct {
		Success   bool   `json:"success"`
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(respBody), "err", err)
		return "", errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
			"worksheetId", worksheetId,
		)
		return "", fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	log.Sugar.Infow("明道云记录创建成功", "worksheetId", worksheetId, "rowId", mdyResp.Data.ID)
	return mdyResp.Data.ID, nil
}

// DeleteRow 删除记录（V3 API）
func (api *MingDaoYunAPI) DeleteRow(worksheetId, rowId string) error {
	path := fmt.Sprintf("/v3/app/worksheets/%s/rows/%s", worksheetId, rowId)

	reqBody := V3DeleteRowRequest{
		TriggerWorkflow: true,
		Permanent:       false,
	}

	respBody, err := api.doV3Request("DELETE", path, reqBody)
	if err != nil {
		return err
	}

	var mdyResp MingDaoYunResponse
	if err := json.Unmarshal(respBody, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(respBody), "err", err)
		return errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
			"rowId", rowId,
		)
		return fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	log.Sugar.Infow("明道云记录删除成功", "worksheetId", worksheetId, "rowId", rowId)
	return nil
}

// ========== 客户微信信息操作 ==========

// CustomerWeComInfo 客户微信信息
type CustomerWeComInfo struct {
	WechatName           string
	WechatGender         string
	WecomExternalUserid  string
	WechatAvatar         string
	WechatUnionId        string
	WecomExternalProfile string
	WecomStaffID         string
}

// UpdateCustomerWeComInfo 更新客户的企微信息到明道云
func (api *MingDaoYunAPI) UpdateCustomerWeComInfo(rowId string, info CustomerWeComInfo) error {
	fields := make(map[string]string)

	if info.WechatName != "" {
		fields["wechatName"] = info.WechatName
	}
	if info.WechatGender != "" {
		fields["wechatGender"] = info.WechatGender
	}
	if info.WecomExternalUserid != "" {
		fields["wecomExternalUserid"] = info.WecomExternalUserid
	}
	if info.WechatAvatar != "" {
		fields["wechatAvatar"] = info.WechatAvatar
	}
	if info.WechatUnionId != "" {
		fields["wechatUnionId"] = info.WechatUnionId
	}
	if info.WecomExternalProfile != "" {
		fields["wecomExternalProfile"] = info.WecomExternalProfile
	}
	if info.WecomStaffID != "" {
		fields["wecomStaffID"] = info.WecomStaffID
	}

	return api.UpdateRow(rowId, fields)
}

// ClearCustomerWeComInfo 清除客户的企微信息
func (api *MingDaoYunAPI) ClearCustomerWeComInfo(rowId string) error {
	fields := map[string]string{
		"wechatName":           "",
		"wechatGender":         "",
		"wecomExternalUserid":  "",
		"wechatAvatar":         "",
		"wechatUnionId":        "",
		"wecomExternalProfile": "",
		"wecomStaffID":         "",
	}
	return api.UpdateRow(rowId, fields)
}

// BindCustomer 绑定客户
func (api *MingDaoYunAPI) BindCustomer(rowId, externalUserID, staffID string) error {
	if rowId == "" || externalUserID == "" {
		return errors.New("rowId 和 externalUserID 不能为空")
	}

	info := CustomerWeComInfo{
		WecomExternalUserid: externalUserID,
	}
	if staffID != "" {
		info.WecomStaffID = staffID
	}

	return api.UpdateCustomerWeComInfo(rowId, info)
}
