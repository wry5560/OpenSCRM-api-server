# 企业微信员工/部门同步到明道云 - 设计文档

> 文档版本: 1.0
> 创建日期: 2026-01-15
> 作者: Claude Code

## 一、概述

### 1.1 背景

OpenSCRM 系统从企业微信同步员工和部门信息后，需要将这些数据同步到明道云工作表，以便在明道云平台进行进一步的数据管理和业务处理。

### 1.2 目标

1. 实现企业微信员工信息到明道云"企微员工信息表"的自动同步
2. 实现企业微信部门信息到明道云"企微部门信息表"的自动同步
3. 修复现有系统中员工添加/删除回调未注册的问题

### 1.3 范围

- 支持全量同步（API 触发）
- 支持增量同步（企微回调触发）
- 异步执行，不阻塞主流程

---

## 二、明道云工作表结构

### 2.1 企微员工信息表

| 属性 | 值 |
|------|-----|
| 工作表ID | `69660cd16951c9bd8f2da03f` |
| 别名 | `qwygxx` |

**字段定义：**

| 字段ID | 字段名 | 别名 | 类型 | 必填 | 说明 |
|--------|--------|------|------|------|------|
| 69660d1cc91f18e9b901afd5 | 企微员工ID | wecom_staff_id | Text | 是 | 标题字段，唯一标识 |
| 69660d1cc91f18e9b901afd7 | 企微用户名 | wecom_username | Text | 是 | 员工姓名 |
| 69660d1cc91f18e9b901afda | 企微头像 | wecom_avatar | Attachment | 否 | 头像图片 |
| 69660d1cc91f18e9b901afdc | 性别 | gender | Dropdown | 否 | 男/女 |
| 69660d1cc91f18e9b901afdf | 手机号码 | phone | PhoneNumber | 否 | |
| 69660d1cc91f18e9b901afe0 | 邮箱 | email | Email | 否 | |
| 69660d1cc91f18e9b901afe3 | 企微部门ID | wecom_dep_id | Relation | 是 | 关联部门表 |
| 69660d1cc91f18e9b901afe7 | 岗位 | position | Text | 否 | |
| 69660d1cc91f18e9b901aff1 | 员工状态 | staff_status | Dropdown | 是 | 在职/试用/离职 |

**下拉选项值：**

```
性别:
  - 男: 27a67e42-741f-43a1-ac68-fa1a752f7373
  - 女: 2bbc0fe0-7cce-4764-9e62-806625a36283

员工状态:
  - 在职: 03084068-0aa5-4c4a-9c6b-37a0d960a877
  - 试用: 8acd41cf-2233-4f07-9af4-5f45e4d1cb8e
  - 离职: 7115d2e2-7d18-4881-b394-9181c395d691
```

### 2.2 企微部门信息表

| 属性 | 值 |
|------|-----|
| 工作表ID | `69660d1cf636abca5cc49905` |
| 别名 | `qwbmxx` |

**字段定义：**

| 字段ID | 字段名 | 类型 | 说明 |
|--------|--------|------|------|
| 69660ddb84223902b9ec7a72 | 部门ID | Text | 标题字段，企微部门ID |
| 69660ddb84223902b9ec7a73 | 部门名 | Text | 部门名称 |

---

## 三、数据映射

### 3.1 部门字段映射

| OpenSCRM 字段 | 类型 | 明道云字段 | 转换规则 |
|--------------|------|-----------|----------|
| Department.ExtID | int64 | 部门ID | `strconv.FormatInt(ExtID, 10)` |
| Department.Name | string | 部门名 | 直接映射 |

### 3.2 员工字段映射

| OpenSCRM 字段 | 类型 | 明道云字段 | 转换规则 |
|--------------|------|-----------|----------|
| Staff.ExtID | string | 企微员工ID | 直接映射（唯一标识） |
| Staff.Name | string | 企微用户名 | 直接映射 |
| Staff.AvatarURL | string | 企微头像 | `[{"url":"${AvatarURL}"}]` |
| Staff.Gender | int | 性别 | 1→男Key, 2→女Key |
| Staff.Mobile | string | 手机号码 | 直接映射 |
| Staff.Email | string | 邮箱 | 直接映射 |
| Staff.DeptIds | []int64 | 企微部门ID | 查询部门 rowid 数组 |
| Staff.Position | string | 岗位 | 直接映射 |
| Staff.Status | int | 员工状态 | 见下表 |

**员工状态映射：**

| OpenSCRM Status | 含义 | 明道云状态 |
|-----------------|------|-----------|
| 1 | 已激活 | 在职 |
| 2 | 已禁用 | 离职 |
| 4 | 未激活 | 试用 |
| 5 | 退出企业 | 离职 |

---

## 四、同步策略

### 4.1 同步触发方式

| 触发方式 | 触发时机 | 同步类型 |
|----------|----------|----------|
| 全量同步 | 调用员工/部门同步 API | 批量 Upsert |
| 增量同步 | 企微回调事件 | 单条 Create/Update/Delete |

### 4.2 同步顺序

**全量同步顺序（必须遵守）：**
1. 先同步所有部门
2. 再同步所有员工

> 原因：员工的"企微部门ID"字段需要关联部门表的 rowid，必须先确保部门记录存在。

**增量同步：**
- 部门事件：直接同步
- 员工事件：同步前检查关联部门是否存在，不存在则先创建

### 4.3 幂等性设计

- **唯一标识**：使用 `ExtID`（企微员工ID/部门ID）作为业务唯一键
- **同步逻辑**：
  1. 根据 ExtID 查询明道云记录
  2. 记录存在 → 更新
  3. 记录不存在 → 创建

### 4.4 同步模式

- **执行方式**：异步执行，使用后台 goroutine
- **错误处理**：失败只记录日志，不阻塞主流程
- **删除策略**：员工删除时标记为"离职"，保留历史记录

---

## 五、技术实现

### 5.1 配置扩展

在 `conf/config.go` 的 `MingDaoYunConfig` 中添加：

```go
type MingDaoYunConfig struct {
    // 现有字段...

    // StaffWorksheetID 员工表工作表ID
    StaffWorksheetID string `json:"staff_worksheet_id" yaml:"staff_worksheet_id"`

    // DepartmentWorksheetID 部门表工作表ID
    DepartmentWorksheetID string `json:"department_worksheet_id" yaml:"department_worksheet_id"`

    // EnableStaffSync 是否启用员工同步到明道云
    EnableStaffSync bool `json:"enable_staff_sync" yaml:"enable_staff_sync"`
}
```

配置文件示例：
```yaml
mingdaoyun:
  staff_worksheet_id: "69660cd16951c9bd8f2da03f"
  department_worksheet_id: "69660d1cf636abca5cc49905"
  enable_staff_sync: true
```

### 5.2 常量定义

在 `app/constants/mingdaoyun.go` 中添加字段映射常量：

```go
// MingDaoYunDepartmentFields 部门表字段映射
var MingDaoYunDepartmentFields = map[string]string{
    "departmentId":   "69660ddb84223902b9ec7a72",
    "departmentName": "69660ddb84223902b9ec7a73",
}

// MingDaoYunStaffFields 员工表字段映射
var MingDaoYunStaffFields = map[string]string{
    "wecomStaffId":  "69660d1cc91f18e9b901afd5",
    "wecomUsername": "69660d1cc91f18e9b901afd7",
    "wecomAvatar":   "69660d1cc91f18e9b901afda",
    "gender":        "69660d1cc91f18e9b901afdc",
    "phone":         "69660d1cc91f18e9b901afdf",
    "email":         "69660d1cc91f18e9b901afe0",
    "wecomDepId":    "69660d1cc91f18e9b901afe3",
    "position":      "69660d1cc91f18e9b901afe7",
    "staffStatus":   "69660d1cc91f18e9b901aff1",
}

// MingDaoYunGenderOptions 性别选项映射
var MingDaoYunGenderOptions = map[int]string{
    1: "27a67e42-741f-43a1-ac68-fa1a752f7373", // 男
    2: "2bbc0fe0-7cce-4764-9e62-806625a36283", // 女
}

// MingDaoYunStaffStatusOptions 员工状态映射
var MingDaoYunStaffStatusOptions = map[int]string{
    1: "03084068-0aa5-4c4a-9c6b-37a0d960a877", // 在职
    2: "7115d2e2-7d18-4881-b394-9181c395d691", // 离职
    4: "8acd41cf-2233-4f07-9af4-5f45e4d1cb8e", // 试用
    5: "7115d2e2-7d18-4881-b394-9181c395d691", // 离职
}
```

### 5.3 核心服务接口

新建 `app/services/mingdaoyun_staff_sync.go`：

```go
// SyncDepartmentToMingDaoYun 同步单个部门到明道云
// action: "create", "update", "delete"
func (s *MingDaoYunService) SyncDepartmentToMingDaoYun(dept *models.Department, action string) error

// SyncStaffToMingDaoYun 同步单个员工到明道云
// action: "create", "update", "delete"
func (s *MingDaoYunService) SyncStaffToMingDaoYun(staff *models.Staff, action string) error

// SyncAllDepartmentsToMingDaoYun 全量同步所有部门
func (s *MingDaoYunService) SyncAllDepartmentsToMingDaoYun(extCorpID string) error

// SyncAllStaffToMingDaoYun 全量同步所有员工
func (s *MingDaoYunService) SyncAllStaffToMingDaoYun(extCorpID string) error

// AsyncSyncWithRetry 异步同步封装（带重试）
func (s *MingDaoYunService) AsyncSyncWithRetry(syncFunc func() error, maxRetries int)
```

---

## 六、回调注册修复

### 6.1 问题描述

在 `app/callback/callback_handler.go` 中，员工添加和删除事件的处理器已实现，但未注册到事件分发器。

**已实现但未注册的处理器：**
- `staff_event.EventAddStaffHandler` - 添加员工
- `staff_event.EventDelStaffHandler` - 删除员工

### 6.2 修复方案

在 `callback_handler.go` 的 `init()` 函数中添加：

```go
// 新建员工事件
services.Event{
    MessageType: workwx.MessageTypeEvent,
    EventType:   workwx.EventTypeChangeContact,
    ChangeType:  workwx.ChangeTypeCreateUser}: staff_event.EventAddStaffHandler,

// 删除员工事件
services.Event{
    MessageType: workwx.MessageTypeEvent,
    EventType:   workwx.EventTypeChangeContact,
    ChangeType:  workwx.ChangeTypeDelUser}: staff_event.EventDelStaffHandler,
```

---

## 七、文件变更清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/callback/callback_handler.go` | 修改 | 注册员工添加/删除事件 |
| `app/constants/mingdaoyun.go` | 修改 | 添加字段映射常量 |
| `conf/config.go` | 修改 | 扩展配置结构体 |
| `conf/config.yaml` | 修改 | 添加新配置项 |
| `app/services/mingdaoyun_api.go` | 修改 | 添加通用 CRUD 方法 |
| `app/services/mingdaoyun_staff_sync.go` | 新建 | 同步业务逻辑 |
| `app/callback/department_event/*.go` | 修改 | 集成明道云同步 |
| `app/callback/staff_event/*.go` | 修改 | 集成明道云同步 |
| `app/services/department.go` | 修改 | 全量同步集成 |
| `app/services/staff.go` | 修改 | 全量同步集成 |

---

## 八、测试验证

### 8.1 回调修复验证

1. 在企业微信后台添加新员工
2. 检查 API 服务日志，确认收到 `create_user` 回调
3. 在企业微信后台删除员工
4. 检查 API 服务日志，确认收到 `delete_user` 回调

### 8.2 部门同步验证

1. 调用部门全量同步 API: `POST /api/v1/staff-admin/department`
2. 使用明道云 API 查询部门表，验证数据正确
3. 在企微后台修改部门名称，验证增量同步

### 8.3 员工同步验证

1. 调用员工全量同步 API: `POST /api/v1/staff-admin/staff`
2. 使用明道云 API 查询员工表，验证数据正确
3. 在企微后台修改员工信息，验证增量同步
4. 在企微后台删除员工，验证状态变为"离职"

---

## 九、风险与注意事项

1. **同步顺序**：全量同步时必须先同步部门，否则员工的部门关联会失败
2. **API 限流**：明道云 API 有请求频率限制，批量同步时需要控制速率
3. **异步执行**：同步在后台执行，失败不会立即反馈给用户
4. **数据一致性**：由于异步执行，可能存在短暂的数据不一致窗口

---

## 十、后续优化

1. 增加同步失败重试队列
2. 增加同步状态监控和告警
3. 支持手动触发单个员工/部门重新同步
4. 增加同步日志记录表
