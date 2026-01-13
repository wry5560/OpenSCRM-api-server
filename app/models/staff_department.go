package models

import (
	"gorm.io/gorm/clause"
	"openscrm/app/constants"
)

type StaffDepartment struct {
	ExtCorpID       string            `json:"ext_corp_id" gorm:"index;type:char(18);uniqueIndex:idx_ext_corp_id_ext_staff_id"`
	ExtStaffID      string            `json:"ext_staff_id" gorm:"type:varchar(64);index;uniqueIndex:idx_ext_corp_id_ext_staff_id"`
	ExtDepartmentID int64             `json:"ext_department_id" gorm:"type:integer;uniqueIndex:idx_ext_corp_id_ext_staff_id"`
	StaffID         string            `json:"staff_id" gorm:"primaryKey;type:bigint" `
	DepartmentID    string            `json:"department_id" gorm:"primaryKey;type:bigint;" `
	IsLeader        constants.Boolean `json:"is_leader" gorm:"type:smallint;comment:是否是所在部门的领导"`
	Order           uint32            `json:"order" gorm:"type:integer;comment:所在部门的排序"`
}

func (s StaffDepartment) Upsert(sd ...StaffDepartment) error {
	// PostgreSQL: 使用复合唯一索引的三个列作为冲突检测
	// 同时去重避免 ON CONFLICT 报错
	uniqueMap := make(map[string]StaffDepartment)
	for _, item := range sd {
		key := item.ExtCorpID + "_" + item.ExtStaffID + "_" + string(rune(item.ExtDepartmentID))
		uniqueMap[key] = item
	}
	uniqueSD := make([]StaffDepartment, 0, len(uniqueMap))
	for _, item := range uniqueMap {
		uniqueSD = append(uniqueSD, item)
	}

	err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ext_corp_id"}, {Name: "ext_staff_id"}, {Name: "ext_department_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"order", "is_leader", "staff_id", "department_id"}),
	}).CreateInBatches(&uniqueSD, len(uniqueSD)).Error
	if err != nil {
		return err
	}

	return err
}

func (s StaffDepartment) Delete(sd ...StaffDepartment) error {
	for _, staffDepartment := range sd {
		err := DB.Model(&StaffDepartment{}).
			Where("ext_corp_id = ? and ext_staff_id = ? and ext_department_id = ?",
				staffDepartment.ExtCorpID, staffDepartment.ExtStaffID, staffDepartment.ExtDepartmentID).Delete(&StaffDepartment{}).Error
		if err != nil {
			return err
		}
	}
	return nil
}
