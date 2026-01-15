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

// MingDaoYunAPI 明道云 API 封装
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

// GetFilterRowsRequest 查询记录列表请求
type GetFilterRowsRequest struct {
	AppKey      string              `json:"appKey"`
	Sign        string              `json:"sign"`
	WorksheetID string              `json:"worksheetId"`
	ViewID      string              `json:"viewId,omitempty"`
	PageSize    int                 `json:"pageSize"`
	PageIndex   int                 `json:"pageIndex"`
	SortID      string              `json:"sortId,omitempty"`
	IsAsc       bool                `json:"isAsc,omitempty"`
	Filters     []FilterCondition   `json:"filters,omitempty"`
	Keywords    string              `json:"keywords,omitempty"`
}

// FilterCondition 过滤条件
type FilterCondition struct {
	ControlID  string   `json:"controlId"`
	DataType   int      `json:"dataType"`
	SpliceType int      `json:"spliceType"`
	FilterType int      `json:"filterType"`
	Value      string   `json:"value,omitempty"`
	Values     []string `json:"values,omitempty"`
}

// GetRowByIDRequest 根据ID获取记录请求
type GetRowByIDRequest struct {
	AppKey      string `json:"appKey"`
	Sign        string `json:"sign"`
	WorksheetID string `json:"worksheetId"`
	RowID       string `json:"rowId"`
}

// CustomerRecord 客户记录
type CustomerRecord struct {
	RowID  string                 `json:"rowid"`
	Fields map[string]interface{} `json:"fields"`
}

// GetFilterRowsResponse 查询记录列表响应
type GetFilterRowsResponse struct {
	Rows  []map[string]interface{} `json:"rows"`
	Total int                      `json:"total"`
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

// UpdateRowRequest 更新行记录请求
type UpdateRowRequest struct {
	AppKey      string             `json:"appkey"`
	Sign        string             `json:"sign"`
	WorksheetID string             `json:"worksheetId"`
	RowID       string             `json:"rowId"`
	Controls    []UpdateRowControl `json:"controls"`
}

// UpdateRowControl 更新行的字段控制
type UpdateRowControl struct {
	ControlID string `json:"controlId"`
	Value     string `json:"value"`
}

// UpdateRow 更新明道云记录
// rowId: 明道云记录 ID (即 userNO)
// fields: 字段映射，key 为字段用途名，value 为字段值
func (api *MingDaoYunAPI) UpdateRow(rowId string, fields map[string]string) error {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return errors.New("明道云配置不完整")
	}

	// 构建控件数组
	controls := make([]UpdateRowControl, 0, len(fields))
	for fieldName, fieldValue := range fields {
		fieldID, ok := constants.MingDaoYunCustomerFields[fieldName]
		if !ok {
			log.Sugar.Warnw("未知的明道云字段", "fieldName", fieldName)
			continue
		}
		controls = append(controls, UpdateRowControl{
			ControlID: fieldID,
			Value:     fieldValue,
		})
	}

	if len(controls) == 0 {
		return errors.New("没有有效的字段需要更新")
	}

	// 构建请求体
	reqBody := UpdateRowRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: constants.MingDaoYunCustomerWorksheetAlias,
		RowID:       rowId,
		Controls:    controls,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrap(err, "序列化请求体失败")
	}

	// 发送请求
	url := fmt.Sprintf("%s/v2/open/worksheet/editRow", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 更新记录",
		"url", url,
		"rowId", rowId,
		"fieldsCount", len(controls),
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "读取响应失败")
	}

	// 解析响应
	var mdyResp MingDaoYunResponse
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
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

	log.Sugar.Infow("明道云记录更新成功", "rowId", rowId)
	return nil
}

// CustomerWeComInfo 客户微信信息（用于更新明道云）
type CustomerWeComInfo struct {
	// WechatName 微信昵称
	WechatName string
	// WechatGender 性别 (1=男, 2=女, 0=未知)
	WechatGender string
	// WecomExternalUserid 企微外部联系人ID
	WecomExternalUserid string
	// WechatAvatar 头像URL
	WechatAvatar string
	// WechatUnionId UnionID
	WechatUnionId string
	// WecomExternalProfile 对外信息JSON
	WecomExternalProfile string
	// WecomStaffID 添加该客户的员工ID
	WecomStaffID string
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

// ClearCustomerWeComInfo 清除客户的企微信息（将所有微信相关字段设为空）
func (api *MingDaoYunAPI) ClearCustomerWeComInfo(rowId string) error {
	fields := make(map[string]string)

	// 将所有微信相关字段设为空字符串
	fields["wechatName"] = ""
	fields["wechatGender"] = ""
	fields["wecomExternalUserid"] = ""
	fields["wechatAvatar"] = ""
	fields["wechatUnionId"] = ""
	fields["wecomExternalProfile"] = ""
	fields["wecomStaffID"] = ""

	return api.UpdateRow(rowId, fields)
}

// GetFilterRows 查询记录列表
func (api *MingDaoYunAPI) GetFilterRows(filters []FilterCondition, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return nil, errors.New("明道云配置不完整")
	}

	reqBody := GetFilterRowsRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: constants.MingDaoYunCustomerWorksheetAlias,
		PageSize:    pageSize,
		PageIndex:   pageIndex,
		Filters:     filters,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/getFilterRows", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 查询记录",
		"url", url,
		"filtersCount", len(filters),
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取响应失败")
	}

	// 解析响应
	var mdyResp struct {
		Success   bool `json:"success"`
		ErrorCode int  `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			Rows  []map[string]interface{} `json:"rows"`
			Total int                      `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
		return nil, errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
		)
		return nil, fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	// 转换为 MingDaoCustomerInfo 列表
	result := &MingDaoCustomerSearchResult{
		Total: mdyResp.Data.Total,
		Items: make([]MingDaoCustomerInfo, 0, len(mdyResp.Data.Rows)),
	}

	for _, row := range mdyResp.Data.Rows {
		rowID, _ := row["rowid"].(string)
		result.Items = append(result.Items, MingDaoCustomerInfo{
			RowID:  rowID,
			Fields: row,
		})
	}

	return result, nil
}

// GetRowByID 根据记录ID获取详情
func (api *MingDaoYunAPI) GetRowByID(rowId string) (*MingDaoCustomerInfo, error) {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return nil, errors.New("明道云配置不完整")
	}

	reqBody := GetRowByIDRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: constants.MingDaoYunCustomerWorksheetAlias,
		RowID:       rowId,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/getRowByIdPost", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 获取记录详情",
		"url", url,
		"rowId", rowId,
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取响应失败")
	}

	// 解析响应
	var mdyResp struct {
		Success   bool                   `json:"success"`
		ErrorCode int                    `json:"error_code"`
		ErrorMsg  string                 `json:"error_msg"`
		Data      map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
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

	// 使用企微外部联系人ID字段进行过滤
	filters := []FilterCondition{
		{
			ControlID:  constants.MingDaoYunCustomerFields["wecomExternalUserid"],
			DataType:   2, // 文本类型
			SpliceType: 1, // AND
			FilterType: 1, // 等于
			Value:      externalUserID,
		},
	}

	result, err := api.GetFilterRows(filters, 1, 1)
	if err != nil {
		return nil, err
	}

	if result.Total == 0 || len(result.Items) == 0 {
		return nil, nil // 未找到匹配的客户
	}

	return &result.Items[0], nil
}

// SearchCustomers 搜索客户（支持手机号、客户编号关键字搜索）
func (api *MingDaoYunAPI) SearchCustomers(keyword string, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	if keyword == "" {
		return nil, errors.New("搜索关键字不能为空")
	}

	// 使用 Keywords 全局搜索，支持搜索手机号、客户编号等字段
	result, err := api.GetFilterRowsWithKeywords(keyword, pageSize, pageIndex)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetFilterRowsWithKeywords 使用关键字搜索记录
func (api *MingDaoYunAPI) GetFilterRowsWithKeywords(keywords string, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return nil, errors.New("明道云配置不完整")
	}

	reqBody := GetFilterRowsRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: constants.MingDaoYunCustomerWorksheetAlias,
		PageSize:    pageSize,
		PageIndex:   pageIndex,
		Keywords:    keywords,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/getFilterRows", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 关键字搜索",
		"url", url,
		"keywords", keywords,
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取响应失败")
	}

	// 解析响应
	var mdyResp struct {
		Success   bool `json:"success"`
		ErrorCode int  `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		Data      struct {
			Rows  []map[string]interface{} `json:"rows"`
			Total int                      `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
		return nil, errors.Wrap(err, "解析响应失败")
	}

	if !mdyResp.Success {
		log.Sugar.Errorw("明道云 API 返回错误",
			"errorCode", mdyResp.ErrorCode,
			"errorMsg", mdyResp.ErrorMsg,
		)
		return nil, fmt.Errorf("明道云 API 错误: %s (code: %d)", mdyResp.ErrorMsg, mdyResp.ErrorCode)
	}

	// 转换结果
	result := &MingDaoCustomerSearchResult{
		Total: mdyResp.Data.Total,
	}
	for _, row := range mdyResp.Data.Rows {
		item := MingDaoCustomerInfo{
			RowID:  fmt.Sprintf("%v", row["rowid"]),
			Fields: row,
		}
		result.Items = append(result.Items, item)
	}

	return result, nil
}

// UpdateCustomerFields 更新客户指定字段
func (api *MingDaoYunAPI) UpdateCustomerFields(rowId string, updates []UpdateRowControl) error {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return errors.New("明道云配置不完整")
	}

	if len(updates) == 0 {
		return errors.New("没有有效的字段需要更新")
	}

	// 验证字段是否可编辑
	editableFields := make(map[string]bool)
	for _, field := range constants.MingDaoYunCustomerDisplayFields {
		if field.Editable {
			editableFields[field.ID] = true
		}
	}

	validUpdates := make([]UpdateRowControl, 0)
	for _, update := range updates {
		if editableFields[update.ControlID] {
			validUpdates = append(validUpdates, update)
		} else {
			log.Sugar.Warnw("尝试更新不可编辑的字段", "controlId", update.ControlID)
		}
	}

	if len(validUpdates) == 0 {
		return errors.New("没有可编辑的字段")
	}

	reqBody := UpdateRowRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: constants.MingDaoYunCustomerWorksheetAlias,
		RowID:       rowId,
		Controls:    validUpdates,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/editRow", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 更新客户字段",
		"url", url,
		"rowId", rowId,
		"fieldsCount", len(validUpdates),
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "读取响应失败")
	}

	var mdyResp MingDaoYunResponse
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
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

	log.Sugar.Infow("客户字段更新成功", "rowId", rowId)
	return nil
}

// BindCustomer 绑定客户（将企微外部联系人ID写入客户记录）
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

// ========== 通用 CRUD 方法 ==========

// CreateRowRequest 创建行记录请求
type CreateRowRequest struct {
	AppKey        string             `json:"appKey"`
	Sign          string             `json:"sign"`
	WorksheetID   string             `json:"worksheetId"`
	Controls      []UpdateRowControl `json:"controls"`
	TriggerWorkflow bool             `json:"triggerWorkflow"`
}

// CreateRowResponse 创建行记录响应
type CreateRowResponse struct {
	Success   bool   `json:"success"`
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
	Data      string `json:"data"` // 新记录的 rowId
}

// CreateRow 创建记录（通用方法，支持指定工作表）
func (api *MingDaoYunAPI) CreateRow(worksheetId string, controls []UpdateRowControl) (string, error) {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return "", errors.New("明道云配置不完整")
	}

	if len(controls) == 0 {
		return "", errors.New("没有有效的字段需要创建")
	}

	reqBody := CreateRowRequest{
		AppKey:          cfg.AppKey,
		Sign:            cfg.Sign,
		WorksheetID:     worksheetId,
		Controls:        controls,
		TriggerWorkflow: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/addRow", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 创建记录",
		"url", url,
		"worksheetId", worksheetId,
		"fieldsCount", len(controls),
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "读取响应失败")
	}

	var mdyResp CreateRowResponse
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
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

	log.Sugar.Infow("明道云记录创建成功", "worksheetId", worksheetId, "rowId", mdyResp.Data)
	return mdyResp.Data, nil
}

// EditRowByWorksheet 更新记录（通用方法，支持指定工作表）
func (api *MingDaoYunAPI) EditRowByWorksheet(worksheetId, rowId string, controls []UpdateRowControl) error {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return errors.New("明道云配置不完整")
	}

	if len(controls) == 0 {
		return errors.New("没有有效的字段需要更新")
	}

	reqBody := UpdateRowRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: worksheetId,
		RowID:       rowId,
		Controls:    controls,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/editRow", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 更新记录",
		"url", url,
		"worksheetId", worksheetId,
		"rowId", rowId,
		"fieldsCount", len(controls),
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "读取响应失败")
	}

	var mdyResp MingDaoYunResponse
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
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

// GetFilterRowsByWorksheet 查询记录列表（通用方法，支持指定工作表）
func (api *MingDaoYunAPI) GetFilterRowsByWorksheet(worksheetId string, filters []FilterCondition, pageSize, pageIndex int) (*MingDaoCustomerSearchResult, error) {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return nil, errors.New("明道云配置不完整")
	}

	reqBody := GetFilterRowsRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: worksheetId,
		PageSize:    pageSize,
		PageIndex:   pageIndex,
		Filters:     filters,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/getFilterRows", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Debugw("调用明道云 API 查询记录",
		"url", url,
		"worksheetId", worksheetId,
		"filtersCount", len(filters),
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取响应失败")
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
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
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
		rowID, _ := row["rowid"].(string)
		result.Items = append(result.Items, MingDaoCustomerInfo{
			RowID:  rowID,
			Fields: row,
		})
	}

	return result, nil
}

// DeleteRowRequest 删除行记录请求
type DeleteRowRequest struct {
	AppKey        string `json:"appKey"`
	Sign          string `json:"sign"`
	WorksheetID   string `json:"worksheetId"`
	RowID         string `json:"rowId"`
	TriggerWorkflow bool `json:"triggerWorkflow"`
}

// DeleteRow 删除记录
func (api *MingDaoYunAPI) DeleteRow(worksheetId, rowId string) error {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return errors.New("明道云配置不完整")
	}

	reqBody := DeleteRowRequest{
		AppKey:          cfg.AppKey,
		Sign:            cfg.Sign,
		WorksheetID:     worksheetId,
		RowID:           rowId,
		TriggerWorkflow: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/deleteRow", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Infow("调用明道云 API 删除记录",
		"url", url,
		"worksheetId", worksheetId,
		"rowId", rowId,
	)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "读取响应失败")
	}

	var mdyResp MingDaoYunResponse
	if err := json.Unmarshal(body, &mdyResp); err != nil {
		log.Sugar.Errorw("解析明道云响应失败", "body", string(body), "err", err)
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
