package constants

// MingDaoYunStatePrefix 明道云二维码 state 前缀
// 用于在企业微信回调中识别来自明道云的添加好友请求
const MingDaoYunStatePrefix = "mdy:"

// ========== 明道云工作表别名常量 ==========
// 使用工作表别名代替 ID，简化配置，避免不同用户/环境的 ID 不一致问题
// 别名在明道云应用内唯一，API 调用时可直接使用别名替代 worksheetId

const (
	// MingDaoYunCustomerWorksheetAlias 客户表别名
	MingDaoYunCustomerWorksheetAlias = "starclientinfo"
	// MingDaoYunStaffWorksheetAlias 员工表别名
	MingDaoYunStaffWorksheetAlias = "starstaffinfo"
	// MingDaoYunDepartmentWorksheetAlias 部门表别名
	MingDaoYunDepartmentWorksheetAlias = "stardeptinfo"
)

// MingDaoYunCustomerFields 明道云客户表字段映射（企微信息绑定时写入）
// Key: 字段用途, Value: 明道云字段别名（使用别名确保迁移一致性）
var MingDaoYunCustomerFields = map[string]string{
	// 企微员工ID
	"wecomStaffID": "wecomStaffID",
	// 微信名
	"wechatName": "wechatName",
	// 微信性别 (1=男, 2=女, 0=未知)
	"wechatGender": "wechatGender",
	// 企微外部联系人ID
	"wecomExternalUserid": "wecomExternalUserid",
	// 微信头像URL
	"wechatAvatar": "wechatAvatar",
	// 微信UnionID
	"wechatUnionId": "wechatUnionId",
	// 企微对外信息(JSON)
	"wecomExternalProfile": "wecomExternalProfile",
}

// MingDaoYunCustomerStaticFields 明道云客户表静态字段配置
// 用于侧边栏客户信息展示和编辑
type CustomerFieldConfig struct {
	ID       string // 字段ID
	Name     string // 字段名称
	Type     string // 字段类型
	Editable bool   // 是否可编辑
}

// MingDaoYunCustomerDisplayFields 侧边栏展示的静态字段列表
var MingDaoYunCustomerDisplayFields = []CustomerFieldConfig{
	{ID: "693660e95326c71216b1b87a", Name: "客户编号", Type: "AutoNumber", Editable: false},
	{ID: "692f976f7001b729cd1c01c1", Name: "号码", Type: "Text", Editable: false},
	{ID: "693660e95326c71216b1b87b", Name: "轨道群号", Type: "Text", Editable: true},
	{ID: "693660e95326c71216b1b87c", Name: "售后群号", Type: "Text", Editable: true},
	{ID: "692f976f7001b729cd1c01bf", Name: "捞客账号", Type: "Text", Editable: true},
	{ID: "692f976f7001b729cd1c01be", Name: "客户抖音名称", Type: "Text", Editable: true},
	{ID: "694a39afa87445aaca8c3ec3", Name: "客户抖音ID", Type: "Text", Editable: true},
	{ID: "692f976f7001b729cd1c01c0", Name: "主播", Type: "Dropdown", Editable: true},
	{ID: "692feedb2328de1fe0c8f600", Name: "日期", Type: "Date", Editable: true},
	{ID: "692f976f7001b729cd1c01c5", Name: "订单编号", Type: "Text", Editable: false},
	{ID: "694e20662a4f51165dfa2264", Name: "收货地址", Type: "Text", Editable: true},
	{ID: "694b6d090d5691f00accd141", Name: "客户意向", Type: "Dropdown", Editable: true},
	{ID: "694b70a80d5691f00acce09f", Name: "客户进度", Type: "Dropdown", Editable: true},
	{ID: "692f976f7001b729cd1c01c2", Name: "需求", Type: "MultipleSelect", Editable: true},
	{ID: "694bc08a0d5691f00ace2e61", Name: "装修进度", Type: "Dropdown", Editable: true},
	{ID: "694bc08a0d5691f00ace2e62", Name: "户型", Type: "Dropdown", Editable: true},
	{ID: "695a258487071723ff4e1dd1", Name: "总收款", Type: "Rollup", Editable: false},
	{ID: "695a258487071723ff4e1dd2", Name: "总退款", Type: "Rollup", Editable: false},
	{ID: "692f976f7001b729cd1c01c3", Name: "设计师", Type: "Collaborator", Editable: false},
	{ID: "694b6c7a0d5691f00acccf70", Name: "轨道", Type: "Collaborator", Editable: false},
}

// MingDaoYunCustomerHiddenFields 需要隐藏的微信相关字段ID列表
var MingDaoYunCustomerHiddenFields = []string{
	"696610f93d7d0e60bca91d26", // 企微员工ID
	"6966103cc62174e0bab32b9c", // 微信名
	"6966103cc62174e0bab32b9d", // 微信性别
	"6966103cc62174e0bab32b9e", // 企微外部ID
	"6966103cc62174e0bab32b9f", // 头像url
	"6966103cc62174e0bab32ba0", // 微信开发平台唯一标识
	"6966103cc62174e0bab32ba1", // 成员对外信息json
	"696613717a7a413b01fc2036", // 嵌入二维码
}

// MingDaoYunDropdownOptions 下拉选项配置
type DropdownOption struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// MingDaoYunFieldOptions 字段选项配置
var MingDaoYunFieldOptions = map[string][]DropdownOption{
	// 主播选项
	"692f976f7001b729cd1c01c0": {
		{Key: "0ac85f4e-180c-4839-9e5f-e17e509f14a4", Value: "老板"},
		{Key: "4335ea70-71a7-4f96-8712-0ff475377573", Value: "熊大"},
		{Key: "bef7b89e-1bfc-4290-9c13-0862f69ccebd", Value: "熊二"},
	},
	// 客户意向选项
	"694b6d090d5691f00accd141": {
		{Key: "10e933d4-fd8a-4d91-8b6e-c4d3e4a5a496", Value: "抖店成交"},
		{Key: "61b00df5-4045-4637-bb06-21725980dab0", Value: "强意向"},
		{Key: "cacdfee2-2072-468d-9038-8a19d36d879a", Value: "潜在"},
		{Key: "011c62f7-abcc-4c7a-96c6-68b6709f3e6c", Value: "沉默"},
		{Key: "a67f2d46-3dd1-4e87-a8a6-09d62c60eb61", Value: "流失"},
	},
	// 客户进度选项
	"694b70a80d5691f00acce09f": {
		{Key: "7a910c03-db3a-4e55-bd19-e33ee6acae21", Value: "添加企微"},
		{Key: "10e933d4-fd8a-4d91-8b6e-c4d3e4a5a496", Value: "已加微信"},
		{Key: "61b00df5-4045-4637-bb06-21725980dab0", Value: "一次跟进"},
		{Key: "011c62f7-abcc-4c7a-96c6-68b6709f3e6c", Value: "二次跟进"},
		{Key: "446bab17-32b5-49ba-8e3b-2eca9a8b5bb3", Value: "三次跟进"},
		{Key: "8e3e8e69-e0fd-485e-825b-c11c1b9cb862", Value: "线下成交"},
	},
	// 需求选项
	"692f976f7001b729cd1c01c2": {
		{Key: "80e0f226-3e6e-441c-be8e-90f72ef74a19", Value: "开关"},
		{Key: "ba2191b5-7e7d-4eb9-8cf1-4fa0b0166ede", Value: "电机"},
		{Key: "be795cd7-4ece-49d6-a591-646a625c2665", Value: "设计师"},
		{Key: "0a56ba1e-77b7-4e30-9646-fb9cc53185b8", Value: "帘布"},
		{Key: "aae6eee7-991e-4db2-a9c0-7e2350ca3df8", Value: "轨道"},
		{Key: "85b49331-5eb6-4c61-94d1-c06bf56c48e7", Value: "全屋智能"},
		{Key: "dedad227-b964-4c92-8489-d7b99e9046b2", Value: "灯具"},
		{Key: "89c8a6c8-46eb-488b-829e-0767201accdc", Value: "暖通"},
		{Key: "309f0c68-4dea-4a56-99de-ed66fab6b34c", Value: "全屋网络"},
	},
	// 装修进度选项
	"694bc08a0d5691f00ace2e61": {
		{Key: "69c0707c-5f61-48a4-82e0-0da31818ae4b", Value: "未进场"},
		{Key: "03c4d3f9-54c9-45d7-9cf7-b73558bf55fc", Value: "水电进场"},
		{Key: "7861e21e-0a9b-4023-8ae1-c564f3941aa2", Value: "木工完成"},
		{Key: "c9584b07-6f2b-4f61-bd6a-af37e31731bc", Value: "油漆完成"},
		{Key: "ba566290-8d32-45fa-a6d6-493684c054f3", Value: "网络完成"},
		{Key: "dd51bc53-0a72-4b80-8934-21916fdd7a20", Value: "已入住"},
	},
	// 户型选项
	"694bc08a0d5691f00ace2e62": {
		{Key: "03c4d3f9-54c9-45d7-9cf7-b73558bf55fc", Value: "一房"},
		{Key: "ada260e7-9df7-4d99-a7f9-105026510354", Value: "两房"},
		{Key: "c4799f90-e77d-4f95-8768-955eec83acb6", Value: "三房"},
		{Key: "dd51bc53-0a72-4b80-8934-21916fdd7a20", Value: "别墅"},
		{Key: "c873ec45-af77-4152-94bd-6a7f30d78086", Value: "复式"},
		{Key: "4d7c60a2-1b52-4630-8cda-941c54909be7", Value: "平层"},
	},
}

// ========== 企微员工/部门同步到明道云相关常量 ==========

// MingDaoYunDepartmentFields 明道云部门表字段映射
// Key: 字段用途, Value: 明道云字段 ID
var MingDaoYunDepartmentFields = map[string]string{
	"departmentId":   "69660ddb84223902b9ec7a72", // 部门ID (标题字段)
	"departmentName": "69660ddb84223902b9ec7a73", // 部门名
}

// MingDaoYunStaffFields 明道云员工表字段映射
// Key: 字段用途, Value: 明道云字段别名（优先使用别名，无别名则使用 ID）
var MingDaoYunStaffFields = map[string]string{
	"wecomStaffId":  "wecom_staff_id", // 企微员工ID (标题字段)
	"wecomUsername": "wecom_username", // 企微用户名
	"wecomAvatar":   "wecom_avatar",   // 企微头像
	"gender":        "gender",         // 性别
	"phone":         "phone",          // 手机号码
	"email":         "email",          // 邮箱
	"wecomDepId":    "wecom_dep_id",   // 企微部门ID (关联字段)
	"position":      "position",       // 岗位
	"staffStatus":   "staff_status",   // 员工状态
}

// MingDaoYunGenderOptions 员工性别选项映射
// Key: OpenSCRM Gender值, Value: 明道云选项Key
var MingDaoYunGenderOptions = map[int]string{
	1: "27a67e42-741f-43a1-ac68-fa1a752f7373", // 男
	2: "2bbc0fe0-7cce-4764-9e62-806625a36283", // 女
}

// MingDaoYunStaffStatusOptions 员工状态选项映射
// Key: OpenSCRM Status值, Value: 明道云选项Key
var MingDaoYunStaffStatusOptions = map[int]string{
	1: "03084068-0aa5-4c4a-9c6b-37a0d960a877", // 在职 (已激活)
	2: "7115d2e2-7d18-4881-b394-9181c395d691", // 离职 (已禁用)
	4: "8acd41cf-2233-4f07-9af4-5f45e4d1cb8e", // 试用 (未激活)
	5: "7115d2e2-7d18-4881-b394-9181c395d691", // 离职 (退出企业)
}
