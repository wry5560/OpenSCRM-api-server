package models

import (
	"gorm.io/gorm"
)

// CustomerRemark 自定义信息
type CustomerRemark struct {
	ExtCorpModel
	Name         string         `gorm:"type:char(64);uniqueIndex:idx_corp_id_name;" json:"name"`
	FieldType    string         `json:"field_type"` // todo rename to remark_type
	HasStaffUsed bool           `json:"has_staff_used"`
	RankNum      int            `gorm:"type:integer" json:"rank_num"`
	Options      []RemarkOption `gorm:"foreignKey:RemarkID" json:"info_option"`
	Timestamp
}

func (r CustomerRemark) ExchangeOrder(id string, id2 string) error {
	// PostgreSQL: 使用事务交换两条记录的 rank_num
	return DB.Transaction(func(tx *gorm.DB) error {
		var a, b CustomerRemark
		if err := tx.Where("id = ?", id).First(&a).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", id2).First(&b).Error; err != nil {
			return err
		}
		if err := tx.Model(&CustomerRemark{}).Where("id = ?", id).Update("rank_num", b.RankNum).Error; err != nil {
			return err
		}
		if err := tx.Model(&CustomerRemark{}).Where("id = ?", id2).Update("rank_num", a.RankNum).Error; err != nil {
			return err
		}
		return nil
	})
}

// RemarkOption 对于多选类型信息的选项
type RemarkOption struct {
	Model
	RemarkID string `json:"remark_id" gorm:"type:bigint;uniqueIndex:idx_remark_id_name"`
	Name     string `json:"name" gorm:"type:char(64);uniqueIndex:idx_remark_id_name"`
	Timestamp
}

func (r CustomerRemark) Create(remark CustomerRemark) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// 创建记录
		if err := tx.Create(&remark).Error; err != nil {
			return err
		}
		// 获取当前最大 rank_num
		var maxRankNum int
		tx.Model(&CustomerRemark{}).Where("ext_corp_id = ?", remark.ExtCorpID).
			Select("COALESCE(MAX(rank_num), 0)").Scan(&maxRankNum)
		// 更新新记录的 rank_num
		return tx.Model(&CustomerRemark{}).Where("id = ?", remark.ID).
			Update("rank_num", maxRankNum+1).Error
	})
}

func (r CustomerRemark) Delete(ids []string, extCorpID string) error {
	return DB.Model(&CustomerRemark{}).Where("ext_corp_id = ?", extCorpID).Where("id in (?)", ids).Delete(&CustomerRemark{}).Error
}

func (r CustomerRemark) Update(remark CustomerRemark) error {
	return DB.Updates(&remark).Error
}

func (r CustomerRemark) Get(extCorpID string) ([]*CustomerRemark, error) {
	var customerRemarks []*CustomerRemark
	if err := DB.Model(&CustomerRemark{}).Preload("Options").Find(&customerRemarks, "ext_corp_id = ?", extCorpID).Error; err != nil {
		return nil, err
	}
	return customerRemarks, nil
}

//------------------------------

func (o RemarkOption) Create(remark RemarkOption) error {
	return DB.Create(&remark).Error
}

func (o RemarkOption) GetTextOption(db *gorm.DB) (*RemarkOption, error) {
	option := &RemarkOption{}
	err := db.Model(&o).First(option).Error
	if err != nil {
		return nil, err
	}
	return option, nil
}

func (o RemarkOption) Update(option *RemarkOption) error {
	return DB.Model(&RemarkOption{}).
		Where("id = ?", option.ID).
		Updates(option).Error
}

func (o RemarkOption) Delete(ids []string) error {
	return DB.Model(&RemarkOption{}).Where("id in (?)", ids).Delete(&RemarkOption{}).Error
}
