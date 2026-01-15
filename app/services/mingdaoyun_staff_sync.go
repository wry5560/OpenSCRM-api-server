package services

import (
	"encoding/json"
	"openscrm/app/constants"
	"openscrm/app/models"
	"openscrm/common/log"
	"openscrm/conf"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// MingDaoYunStaffSyncService 明道云员工同步服务
type MingDaoYunStaffSyncService struct {
	api *MingDaoYunAPI
}

// NewMingDaoYunStaffSyncService 创建明道云员工同步服务
func NewMingDaoYunStaffSyncService() *MingDaoYunStaffSyncService {
	return &MingDaoYunStaffSyncService{
		api: NewMingDaoYunAPI(),
	}
}

// IsEnabled 检查是否启用员工同步
// 使用硬编码的工作表别名，只需检查 API 配置和开关
func (s *MingDaoYunStaffSyncService) IsEnabled() bool {
	cfg := conf.Settings.MingDaoYun
	return cfg.EnableStaffSync && cfg.AppKey != "" && cfg.Sign != ""
}

// ========== 部门同步 ==========

// SyncDepartmentToMingDaoYun 同步单个部门到明道云
// action: "create", "update", "delete"
func (s *MingDaoYunStaffSyncService) SyncDepartmentToMingDaoYun(dept *models.Department, action string) error {
	if !s.IsEnabled() {
		log.Sugar.Debugw("明道云员工同步未启用，跳过部门同步")
		return nil
	}

	if dept == nil {
		return errors.New("部门对象为空")
	}

	deptExtID := strconv.FormatInt(dept.ExtID, 10)

	log.Sugar.Infow("开始同步部门到明道云",
		"action", action,
		"deptExtId", deptExtID,
		"deptName", dept.Name,
	)

	switch action {
	case "create", "update":
		// 先查询是否存在
		existingRow, err := s.findDepartmentByExtID(deptExtID)
		if err != nil {
			log.Sugar.Warnw("查询部门记录失败", "err", err, "deptExtId", deptExtID)
		}

		controls := s.buildDepartmentControls(dept)

		if existingRow != nil {
			// 存在则更新
			err = s.api.EditRowByWorksheet(constants.MingDaoYunDepartmentWorksheetAlias, existingRow.RowID, controls)
			if err != nil {
				return errors.Wrap(err, "更新部门记录失败")
			}
			log.Sugar.Infow("部门记录更新成功", "deptExtId", deptExtID, "rowId", existingRow.RowID)
		} else {
			// 不存在则创建
			rowId, err := s.api.CreateRow(constants.MingDaoYunDepartmentWorksheetAlias, controls)
			if err != nil {
				return errors.Wrap(err, "创建部门记录失败")
			}
			log.Sugar.Infow("部门记录创建成功", "deptExtId", deptExtID, "rowId", rowId)
		}

	case "delete":
		// 查找记录
		existingRow, err := s.findDepartmentByExtID(deptExtID)
		if err != nil || existingRow == nil {
			log.Sugar.Warnw("部门记录不存在，无需删除", "deptExtId", deptExtID)
			return nil
		}

		// 物理删除部门记录
		err = s.api.DeleteRow(constants.MingDaoYunDepartmentWorksheetAlias, existingRow.RowID)
		if err != nil {
			return errors.Wrap(err, "删除部门记录失败")
		}
		log.Sugar.Infow("部门记录删除成功", "deptExtId", deptExtID, "rowId", existingRow.RowID)

	default:
		return errors.Errorf("未知的同步动作: %s", action)
	}

	return nil
}

// buildDepartmentControls 构建部门字段控件
func (s *MingDaoYunStaffSyncService) buildDepartmentControls(dept *models.Department) []UpdateRowControl {
	controls := []UpdateRowControl{
		{
			ControlID: constants.MingDaoYunDepartmentFields["departmentId"],
			Value:     strconv.FormatInt(dept.ExtID, 10),
		},
		{
			ControlID: constants.MingDaoYunDepartmentFields["departmentName"],
			Value:     dept.Name,
		},
	}
	return controls
}

// findDepartmentByExtID 根据企微部门ID查找明道云记录
func (s *MingDaoYunStaffSyncService) findDepartmentByExtID(extID string) (*MingDaoCustomerInfo, error) {
	filters := []FilterCondition{
		{
			ControlID:  constants.MingDaoYunDepartmentFields["departmentId"],
			DataType:   2,  // 文本类型
			SpliceType: 1,  // AND
			FilterType: 1,  // 等于
			Value:      extID,
		},
	}

	result, err := s.api.GetFilterRowsByWorksheet(constants.MingDaoYunDepartmentWorksheetAlias, filters, 1, 1)
	if err != nil {
		return nil, err
	}

	if result.Total == 0 || len(result.Items) == 0 {
		return nil, nil
	}

	return &result.Items[0], nil
}

// ========== 员工同步 ==========

// SyncStaffToMingDaoYun 同步单个员工到明道云
// action: "create", "update", "delete"
func (s *MingDaoYunStaffSyncService) SyncStaffToMingDaoYun(staff *models.Staff, action string) error {
	if !s.IsEnabled() {
		log.Sugar.Debugw("明道云员工同步未启用，跳过员工同步")
		return nil
	}

	if staff == nil {
		return errors.New("员工对象为空")
	}

	log.Sugar.Infow("开始同步员工到明道云",
		"action", action,
		"staffExtId", staff.ExtID,
		"staffName", staff.Name,
	)

	switch action {
	case "create", "update":
		// 先查询是否存在
		existingRow, err := s.findStaffByExtID(staff.ExtID)
		if err != nil {
			log.Sugar.Warnw("查询员工记录失败", "err", err, "staffExtId", staff.ExtID)
		}

		// 获取部门关联的rowid
		deptRowIds, err := s.getDepartmentRowIds(staff.DeptIds, staff.ExtCorpID)
		if err != nil {
			log.Sugar.Warnw("获取部门rowid失败", "err", err, "deptIds", staff.DeptIds)
		}

		controls := s.buildStaffControls(staff, deptRowIds)

		if existingRow != nil {
			// 存在则更新
			err = s.api.EditRowByWorksheet(constants.MingDaoYunStaffWorksheetAlias, existingRow.RowID, controls)
			if err != nil {
				return errors.Wrap(err, "更新员工记录失败")
			}
			log.Sugar.Infow("员工记录更新成功", "staffExtId", staff.ExtID, "rowId", existingRow.RowID)
		} else {
			// 不存在则创建
			rowId, err := s.api.CreateRow(constants.MingDaoYunStaffWorksheetAlias, controls)
			if err != nil {
				return errors.Wrap(err, "创建员工记录失败")
			}
			log.Sugar.Infow("员工记录创建成功", "staffExtId", staff.ExtID, "rowId", rowId)
		}

	case "delete":
		// 员工删除时不物理删除，而是将状态改为"离职"
		existingRow, err := s.findStaffByExtID(staff.ExtID)
		if err != nil || existingRow == nil {
			log.Sugar.Warnw("员工记录不存在，无需更新状态", "staffExtId", staff.ExtID)
			return nil
		}

		// 更新状态为离职
		statusKey := constants.MingDaoYunStaffStatusOptions[2] // 离职状态
		controls := []UpdateRowControl{
			{
				ControlID: constants.MingDaoYunStaffFields["staffStatus"],
				Value:     buildDropdownValue(statusKey),
			},
		}

		err = s.api.EditRowByWorksheet(constants.MingDaoYunStaffWorksheetAlias, existingRow.RowID, controls)
		if err != nil {
			return errors.Wrap(err, "更新员工离职状态失败")
		}
		log.Sugar.Infow("员工状态已更新为离职", "staffExtId", staff.ExtID, "rowId", existingRow.RowID)

	default:
		return errors.Errorf("未知的同步动作: %s", action)
	}

	return nil
}

// buildStaffControls 构建员工字段控件
func (s *MingDaoYunStaffSyncService) buildStaffControls(staff *models.Staff, deptRowIds []string) []UpdateRowControl {
	controls := []UpdateRowControl{
		{
			ControlID: constants.MingDaoYunStaffFields["wecomStaffId"],
			Value:     staff.ExtID,
		},
		{
			ControlID: constants.MingDaoYunStaffFields["wecomUsername"],
			Value:     staff.Name,
		},
	}

	// 头像（附件格式）
	if staff.AvatarURL != "" {
		avatarValue := buildAttachmentValue(staff.AvatarURL)
		controls = append(controls, UpdateRowControl{
			ControlID: constants.MingDaoYunStaffFields["wecomAvatar"],
			Value:     avatarValue,
		})
	}

	// 性别
	if genderKey, ok := constants.MingDaoYunGenderOptions[int(staff.Gender)]; ok {
		controls = append(controls, UpdateRowControl{
			ControlID: constants.MingDaoYunStaffFields["gender"],
			Value:     buildDropdownValue(genderKey),
		})
	}

	// 手机号码
	if staff.Mobile != "" {
		controls = append(controls, UpdateRowControl{
			ControlID: constants.MingDaoYunStaffFields["phone"],
			Value:     staff.Mobile,
		})
	}

	// 邮箱
	if staff.Email != "" {
		controls = append(controls, UpdateRowControl{
			ControlID: constants.MingDaoYunStaffFields["email"],
			Value:     staff.Email,
		})
	}

	// 部门关联
	if len(deptRowIds) > 0 {
		controls = append(controls, UpdateRowControl{
			ControlID: constants.MingDaoYunStaffFields["wecomDepId"],
			Value:     buildRelationValue(deptRowIds),
		})
	}

	// 岗位
	if staff.ExternalPosition != "" {
		controls = append(controls, UpdateRowControl{
			ControlID: constants.MingDaoYunStaffFields["position"],
			Value:     staff.ExternalPosition,
		})
	}

	// 员工状态
	if statusKey, ok := constants.MingDaoYunStaffStatusOptions[int(staff.Status)]; ok {
		controls = append(controls, UpdateRowControl{
			ControlID: constants.MingDaoYunStaffFields["staffStatus"],
			Value:     buildDropdownValue(statusKey),
		})
	}

	return controls
}

// findStaffByExtID 根据企微员工ID查找明道云记录
func (s *MingDaoYunStaffSyncService) findStaffByExtID(extID string) (*MingDaoCustomerInfo, error) {
	filters := []FilterCondition{
		{
			ControlID:  constants.MingDaoYunStaffFields["wecomStaffId"],
			DataType:   2,  // 文本类型
			SpliceType: 1,  // AND
			FilterType: 1,  // 等于
			Value:      extID,
		},
	}

	result, err := s.api.GetFilterRowsByWorksheet(constants.MingDaoYunStaffWorksheetAlias, filters, 1, 1)
	if err != nil {
		return nil, err
	}

	if result.Total == 0 || len(result.Items) == 0 {
		return nil, nil
	}

	return &result.Items[0], nil
}

// getDepartmentRowIds 获取部门在明道云中的rowid列表
func (s *MingDaoYunStaffSyncService) getDepartmentRowIds(deptIds constants.Int64ArrayField, extCorpID string) ([]string, error) {
	rowIds := make([]string, 0, len(deptIds))

	for _, deptId := range deptIds {
		deptExtID := strconv.FormatInt(deptId, 10)
		deptRow, err := s.findDepartmentByExtID(deptExtID)
		if err != nil {
			log.Sugar.Warnw("查找部门失败", "deptExtId", deptExtID, "err", err)
			continue
		}
		if deptRow != nil {
			rowIds = append(rowIds, deptRow.RowID)
		} else {
			// 部门不存在，尝试先同步该部门
			log.Sugar.Warnw("部门在明道云中不存在，尝试创建", "deptExtId", deptExtID)
			dept, err := models.Department{}.GetByExtID(deptId, extCorpID)
			if err == nil {
				if syncErr := s.SyncDepartmentToMingDaoYun(&dept, "create"); syncErr == nil {
					// 创建成功后重新查找
					newDeptRow, _ := s.findDepartmentByExtID(deptExtID)
					if newDeptRow != nil {
						rowIds = append(rowIds, newDeptRow.RowID)
					}
				}
			}
		}
	}

	return rowIds, nil
}

// ========== 全量同步 ==========

// SyncAllDepartmentsToMingDaoYun 全量同步所有部门到明道云
func (s *MingDaoYunStaffSyncService) SyncAllDepartmentsToMingDaoYun(extCorpID string) error {
	if !s.IsEnabled() {
		log.Sugar.Infow("明道云员工同步未启用，跳过全量部门同步")
		return nil
	}

	log.Sugar.Infow("开始全量同步部门到明道云", "extCorpID", extCorpID)

	// 查询所有部门
	var departments []models.Department
	err := models.DB.Where("ext_corp_id = ?", extCorpID).Find(&departments).Error
	if err != nil {
		return errors.Wrap(err, "查询部门列表失败")
	}

	log.Sugar.Infow("查询到部门数量", "count", len(departments))

	successCount := 0
	failCount := 0

	for _, dept := range departments {
		err := s.SyncDepartmentToMingDaoYun(&dept, "update")
		if err != nil {
			log.Sugar.Errorw("同步部门失败",
				"deptExtId", dept.ExtID,
				"deptName", dept.Name,
				"err", err,
			)
			failCount++
		} else {
			successCount++
		}

		// 避免请求过快触发限流
		time.Sleep(100 * time.Millisecond)
	}

	log.Sugar.Infow("全量部门同步完成",
		"total", len(departments),
		"success", successCount,
		"fail", failCount,
	)

	return nil
}

// SyncAllStaffToMingDaoYun 全量同步所有员工到明道云
func (s *MingDaoYunStaffSyncService) SyncAllStaffToMingDaoYun(extCorpID string) error {
	if !s.IsEnabled() {
		log.Sugar.Infow("明道云员工同步未启用，跳过全量员工同步")
		return nil
	}

	log.Sugar.Infow("开始全量同步员工到明道云", "extCorpID", extCorpID)

	// 查询所有员工
	var staffList []models.Staff
	err := models.DB.Where("ext_corp_id = ?", extCorpID).Find(&staffList).Error
	if err != nil {
		return errors.Wrap(err, "查询员工列表失败")
	}

	log.Sugar.Infow("查询到员工数量", "count", len(staffList))

	successCount := 0
	failCount := 0

	for _, staff := range staffList {
		err := s.SyncStaffToMingDaoYun(&staff, "update")
		if err != nil {
			log.Sugar.Errorw("同步员工失败",
				"staffExtId", staff.ExtID,
				"staffName", staff.Name,
				"err", err,
			)
			failCount++
		} else {
			successCount++
		}

		// 避免请求过快触发限流
		time.Sleep(100 * time.Millisecond)
	}

	log.Sugar.Infow("全量员工同步完成",
		"total", len(staffList),
		"success", successCount,
		"fail", failCount,
	)

	return nil
}

// ========== 异步同步封装 ==========

// AsyncSyncDepartment 异步同步部门（带重试）
func (s *MingDaoYunStaffSyncService) AsyncSyncDepartment(dept *models.Department, action string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Sugar.Errorw("异步同步部门发生panic", "recover", r, "deptExtId", dept.ExtID)
			}
		}()

		s.syncWithRetry(func() error {
			return s.SyncDepartmentToMingDaoYun(dept, action)
		}, 3, "部门", strconv.FormatInt(dept.ExtID, 10))
	}()
}

// AsyncSyncStaff 异步同步员工（带重试）
func (s *MingDaoYunStaffSyncService) AsyncSyncStaff(staff *models.Staff, action string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Sugar.Errorw("异步同步员工发生panic", "recover", r, "staffExtId", staff.ExtID)
			}
		}()

		s.syncWithRetry(func() error {
			return s.SyncStaffToMingDaoYun(staff, action)
		}, 3, "员工", staff.ExtID)
	}()
}

// AsyncSyncAllDepartments 异步全量同步所有部门
func (s *MingDaoYunStaffSyncService) AsyncSyncAllDepartments(extCorpID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Sugar.Errorw("异步全量同步部门发生panic", "recover", r)
			}
		}()

		if err := s.SyncAllDepartmentsToMingDaoYun(extCorpID); err != nil {
			log.Sugar.Errorw("异步全量同步部门失败", "err", err)
		}
	}()
}

// AsyncSyncAllStaff 异步全量同步所有员工
func (s *MingDaoYunStaffSyncService) AsyncSyncAllStaff(extCorpID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Sugar.Errorw("异步全量同步员工发生panic", "recover", r)
			}
		}()

		if err := s.SyncAllStaffToMingDaoYun(extCorpID); err != nil {
			log.Sugar.Errorw("异步全量同步员工失败", "err", err)
		}
	}()
}

// syncWithRetry 带重试的同步封装
func (s *MingDaoYunStaffSyncService) syncWithRetry(syncFunc func() error, maxRetries int, entityType, entityID string) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// 重试前等待，指数退避
			waitTime := time.Duration(i*i) * time.Second
			log.Sugar.Infow("重试同步",
				"entityType", entityType,
				"entityID", entityID,
				"attempt", i+1,
				"waitTime", waitTime,
			)
			time.Sleep(waitTime)
		}

		lastErr = syncFunc()
		if lastErr == nil {
			return
		}

		log.Sugar.Warnw("同步失败，准备重试",
			"entityType", entityType,
			"entityID", entityID,
			"attempt", i+1,
			"err", lastErr,
		)
	}

	log.Sugar.Errorw("同步最终失败",
		"entityType", entityType,
		"entityID", entityID,
		"maxRetries", maxRetries,
		"lastErr", lastErr,
	)
}

// ========== 辅助函数 ==========

// buildDropdownValue 构建下拉选项值的JSON字符串
func buildDropdownValue(optionKey string) string {
	value, _ := json.Marshal([]string{optionKey})
	return string(value)
}

// buildRelationValue 构建关联字段值的JSON字符串
func buildRelationValue(rowIds []string) string {
	value, _ := json.Marshal(rowIds)
	return string(value)
}

// buildAttachmentValue 构建附件字段值的JSON字符串
// V3 API 需要同时包含 name 和 url 字段
func buildAttachmentValue(url string) string {
	// 从URL中提取文件名，如果无法提取则使用默认名称
	fileName := "avatar.jpg"
	if url != "" {
		// 尝试从URL中提取文件名
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			// 移除查询参数
			if idx := strings.Index(lastPart, "?"); idx > 0 {
				lastPart = lastPart[:idx]
			}
			if lastPart != "" && strings.Contains(lastPart, ".") {
				fileName = lastPart
			}
		}
	}

	attachments := []map[string]string{
		{
			"name": fileName,
			"url":  url,
		},
	}
	value, _ := json.Marshal(attachments)
	return string(value)
}
