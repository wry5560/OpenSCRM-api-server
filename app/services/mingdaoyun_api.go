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
// 动态获取视图中的可编辑字段进行验证
func (api *MingDaoYunAPI) UpdateCustomerFields(rowId string, updates []UpdateRowControl) error {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return errors.New("明道云配置不完整")
	}

	if len(updates) == 0 {
		return errors.New("没有有效的字段需要更新")
	}

	// 动态获取视图中的可编辑字段
	viewFields, err := api.GetViewFields(
		constants.MingDaoYunCustomerWorksheetAlias,
		constants.MingDaoYunCustomerSidebarViewID,
	)
	if err != nil {
		log.Sugar.Warnw("获取视图字段失败，跳过字段可编辑性验证", "err", err)
		// 如果获取失败，直接提交更新（由明道云后端验证权限）
		viewFields = nil
	}

	var validUpdates []UpdateRowControl
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

		// 验证更新字段
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
	} else {
		// 没有视图字段信息时，直接使用所有更新
		validUpdates = updates
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

// ========== 视图字段动态获取相关 ==========

// ViewField 视图字段结构（用于前端动态渲染）
type ViewField struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Alias    string        `json:"alias,omitempty"`
	Type     string        `json:"type"`
	SubType  int           `json:"subType,omitempty"`
	Options  []FieldOption `json:"options,omitempty"`
	Required bool          `json:"required"`
	Editable bool          `json:"editable"`
	IsTitle  bool          `json:"isTitle,omitempty"`
	Unit     string        `json:"unit,omitempty"`
	Precision int          `json:"precision,omitempty"`
}

// FieldOption 字段选项
type FieldOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index int    `json:"index,omitempty"`
}

// WorksheetInfoResponse 工作表结构响应
type WorksheetInfoResponse struct {
	WorksheetID string      `json:"worksheetId"`
	Name        string      `json:"name"`
	Views       []ViewInfo  `json:"views"`
	Controls    []ControlInfo `json:"controls"`
}

// ViewInfo 视图信息
type ViewInfo struct {
	ViewID   string   `json:"viewId"`
	Name     string   `json:"name"`
	ViewType int      `json:"viewType"`
	Controls []string `json:"controls,omitempty"` // 视图包含的字段ID列表
}

// ControlInfo 控件/字段信息
type ControlInfo struct {
	ControlID   string          `json:"controlId"`
	ControlName string          `json:"controlName"`
	Type        int             `json:"type"`
	Alias       string          `json:"alias,omitempty"`
	Required    bool            `json:"required"`
	Options     []ControlOption `json:"options,omitempty"`
	Unit        string          `json:"unit,omitempty"`
	Dot         int             `json:"dot,omitempty"` // 小数位数
	Attribute   int             `json:"attribute,omitempty"`
}

// ControlOption 控件选项
type ControlOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index int    `json:"index"`
}

// GetWorksheetInfoRequest 获取工作表结构请求
type GetWorksheetInfoRequest struct {
	AppKey      string `json:"appKey"`
	Sign        string `json:"sign"`
	WorksheetID string `json:"worksheetId"`
}

// GetWorksheetInfo 获取工作表结构（包含字段和视图信息）
func (api *MingDaoYunAPI) GetWorksheetInfo(worksheetId string) (*WorksheetInfoResponse, error) {
	cfg := conf.Settings.MingDaoYun
	if cfg.APIBase == "" || cfg.AppKey == "" || cfg.Sign == "" {
		return nil, errors.New("明道云配置不完整")
	}

	reqBody := GetWorksheetInfoRequest{
		AppKey:      cfg.AppKey,
		Sign:        cfg.Sign,
		WorksheetID: worksheetId,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "序列化请求体失败")
	}

	url := fmt.Sprintf("%s/v2/open/worksheet/getWorksheetInfo", cfg.APIBase)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	log.Sugar.Debugw("调用明道云 API 获取工作表结构",
		"url", url,
		"worksheetId", worksheetId,
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
			WorksheetID string `json:"worksheetId"`
			Name        string `json:"name"`
			Views       []struct {
				ViewID   string   `json:"viewId"`
				Name     string   `json:"name"`
				ViewType int      `json:"viewType"`
				Controls []string `json:"controls"`
			} `json:"views"`
			Controls []struct {
				ControlID   string `json:"controlId"`
				ControlName string `json:"controlName"`
				Type        int    `json:"type"`
				Alias       string `json:"alias"`
				Required    bool   `json:"required"`
				Options     []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
					Index int    `json:"index"`
				} `json:"options"`
				Unit      string `json:"unit"`
				Dot       int    `json:"dot"`
				Attribute int    `json:"attribute"`
			} `json:"controls"`
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

	// 转换为内部结构
	result := &WorksheetInfoResponse{
		WorksheetID: mdyResp.Data.WorksheetID,
		Name:        mdyResp.Data.Name,
	}

	// 转换视图信息
	for _, v := range mdyResp.Data.Views {
		result.Views = append(result.Views, ViewInfo{
			ViewID:   v.ViewID,
			Name:     v.Name,
			ViewType: v.ViewType,
			Controls: v.Controls,
		})
	}

	// 转换控件信息
	for _, c := range mdyResp.Data.Controls {
		ctrl := ControlInfo{
			ControlID:   c.ControlID,
			ControlName: c.ControlName,
			Type:        c.Type,
			Alias:       c.Alias,
			Required:    c.Required,
			Unit:        c.Unit,
			Dot:         c.Dot,
			Attribute:   c.Attribute,
		}
		for _, opt := range c.Options {
			ctrl.Options = append(ctrl.Options, ControlOption{
				Key:   opt.Key,
				Value: opt.Value,
				Index: opt.Index,
			})
		}
		result.Controls = append(result.Controls, ctrl)
	}

	return result, nil
}

// GetViewFields 获取指定视图的字段列表（用于前端动态渲染）
// viewId: 视图ID
// 返回按视图配置顺序排列的字段列表，包含类型、选项等信息
func (api *MingDaoYunAPI) GetViewFields(worksheetId, viewId string) ([]ViewField, error) {
	// 获取工作表结构
	wsInfo, err := api.GetWorksheetInfo(worksheetId)
	if err != nil {
		return nil, errors.Wrap(err, "获取工作表结构失败")
	}

	// 查找目标视图
	var targetView *ViewInfo
	for i := range wsInfo.Views {
		if wsInfo.Views[i].ViewID == viewId {
			targetView = &wsInfo.Views[i]
			break
		}
	}

	if targetView == nil {
		return nil, fmt.Errorf("未找到视图: %s", viewId)
	}

	// 构建控件ID到控件信息的映射
	controlMap := make(map[string]*ControlInfo)
	for i := range wsInfo.Controls {
		controlMap[wsInfo.Controls[i].ControlID] = &wsInfo.Controls[i]
	}

	// 按视图中的字段顺序构建结果
	var fields []ViewField
	for _, ctrlID := range targetView.Controls {
		ctrl, ok := controlMap[ctrlID]
		if !ok {
			continue
		}

		field := ViewField{
			ID:        ctrl.ControlID,
			Name:      ctrl.ControlName,
			Alias:     ctrl.Alias,
			Type:      controlTypeToString(ctrl.Type),
			SubType:   ctrl.Attribute,
			Required:  ctrl.Required,
			Editable:  isFieldEditable(ctrl.Type),
			Unit:      ctrl.Unit,
			Precision: ctrl.Dot,
		}

		// 转换选项
		for _, opt := range ctrl.Options {
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
		"viewName", targetView.Name,
		"fieldsCount", len(fields),
	)

	return fields, nil
}

// controlTypeToString 将明道云控件类型数字转换为字符串
func controlTypeToString(typeNum int) string {
	typeMap := map[int]string{
		1:  "Text",           // 文本
		2:  "Text",           // 文本(多行)
		3:  "Phone",          // 手机号码
		4:  "Phone",          // 座机
		5:  "Email",          // 邮箱
		6:  "Number",         // 数值
		7:  "Money",          // 金额
		8:  "Date",           // 日期
		9:  "Area",           // 地区
		10: "MultipleSelect", // 多选
		11: "Dropdown",       // 单选
		14: "Attachment",     // 附件
		15: "DateTime",       // 日期时间
		16: "DateTime",       // 日期时间范围
		19: "Area",           // 地区(省市县)
		20: "Relation",       // 关联记录
		21: "Relation",       // 关联记录(多条)
		22: "Divider",        // 分段
		23: "Rich",           // 富文本
		24: "Location",       // 定位
		25: "AutoNumber",     // 自动编号
		26: "Collaborator",   // 成员
		27: "Department",     // 部门
		28: "Rating",         // 等级
		29: "Relation",       // 关联查询
		30: "Check",          // 他表字段
		31: "Formula",        // 公式
		32: "Text",           // 文本组合
		33: "Switch",         // 开关
		34: "SubTable",       // 子表
		35: "Cascader",       // 级联选择
		36: "Switch",         // 检查项
		37: "Rollup",         // 汇总
		38: "Formula",        // 公式(日期)
		40: "Location",       // 定位(多个)
		41: "Rich",           // 富文本(嵌入)
		42: "Signature",      // 签名
		43: "OrgRole",        // 组织角色
		45: "Embed",          // 嵌入
		46: "DateTime",       // 时间
		47: "BarCode",        // 条码
		48: "QueryRecord",    // 查询记录
		49: "Section",        // 标签页
		50: "API",            // API查询
		51: "Approval",       // 审批流程
		52: "Section",        // 分组
	}

	if typeName, ok := typeMap[typeNum]; ok {
		return typeName
	}
	return fmt.Sprintf("Unknown(%d)", typeNum)
}

// isFieldEditable 判断字段是否可编辑
func isFieldEditable(typeNum int) bool {
	// 不可编辑的类型
	nonEditableTypes := map[int]bool{
		25: true, // 自动编号
		29: true, // 关联查询
		30: true, // 他表字段
		31: true, // 公式
		37: true, // 汇总
		38: true, // 日期公式
		48: true, // 查询记录
		51: true, // 审批流程
	}

	// 布局类型也不可编辑
	layoutTypes := map[int]bool{
		22: true, // 分段
		41: true, // 嵌入
		45: true, // 嵌入
		49: true, // 标签页
		52: true, // 分组
	}

	return !nonEditableTypes[typeNum] && !layoutTypes[typeNum]
}
