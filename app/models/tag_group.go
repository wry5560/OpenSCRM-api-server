package models

import (
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"openscrm/app/constants"
	"openscrm/app/requests"
	"openscrm/common/app"
)

type TagGroup struct {
	ExtCorpModel
	ExtID          string                    `gorm:"type:char(64);uniqueIndex;comment:外部标签分组ID" json:"ext_id"`
	Name           string                    `gorm:"index;comment:组名字" json:"name"`
	CreateTime     int                       `gorm:"type:integer;comment:" json:"create_time"`
	Order          uint32                    `gorm:"type:integer;index;comment:order值大的排序靠前" json:"order"`
	DepartmentList constants.Int64ArrayField `gorm:"type:jsonb;comment:该标签组可用部门列表,默认0全部可用;" json:"department_list"`
	Tags           []Tag                     `gorm:"foreignKey:ExtGroupID;references:ExtID" json:"tags"`
	Timestamp
}

func (tg TagGroup) ExchangeOrder(ID string, ID2 string) error {
	// PostgreSQL: 使用事务交换两条记录的 order 和 updated_at
	return DB.Transaction(func(tx *gorm.DB) error {
		var a, b TagGroup
		if err := tx.Where("id = ?", ID).First(&a).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", ID2).First(&b).Error; err != nil {
			return err
		}
		if err := tx.Model(&TagGroup{}).Where("id = ?", ID).Updates(map[string]interface{}{
			"order":      b.Order,
			"updated_at": b.UpdatedAt,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&TagGroup{}).Where("id = ?", ID2).Updates(map[string]interface{}{
			"order":      a.Order,
			"updated_at": a.UpdatedAt,
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

// Query
// Description: Query 标签组
// Detail: 按关键词和可用部门搜索
func (tg TagGroup) Query(param requests.TagListReq, extCorpID string) (groups []*TagGroup, total int64, err error) {
	// 查找指定部门
	matchedTagGroupIDs := make([]string, 0)
	shouldMatch := false
	if len(param.ExtDepartmentIDs) > 0 && !funk.ContainsInt64(param.ExtDepartmentIDs, 0) {
		shouldMatch = true
		allTagGroups := make([]TagGroup, 0)
		err = DB.Model(&TagGroup{}).Select("id,department_list").Where("ext_corp_id = ?", extCorpID).Find(&allTagGroups).Error
		if err != nil {
			err = errors.WithStack(err)
			return
		}

		for _, tagGroup := range allTagGroups {

			if tagGroup.DepartmentList == nil || len(tagGroup.DepartmentList) == 0 || funk.ContainsInt64(tagGroup.DepartmentList, 0) {
				// 全局可见的标签组默认不进入指定了部门的搜索结果
				//matchedTagGroupIDs = append(matchedTagGroupIDs, tagGroup.ID)
				continue
			}

			for _, extDepartmentID := range param.ExtDepartmentIDs {
				if funk.ContainsInt64(tagGroup.DepartmentList, extDepartmentID) {
					matchedTagGroupIDs = append(matchedTagGroupIDs, tagGroup.ID)
					continue
				}
			}
		}
	}

	db := DB.Table("tag_group").Joins(" join tag on tag.ext_group_id = tag_group.ext_id ").Where("tag_group.ext_corp_id = ?", extCorpID)
	if shouldMatch {
		db = db.Where("tag_group.id in (?)", matchedTagGroupIDs)
	}

	if param.Name != "" {
		db = db.Where("tag_group.name like ? or tag.name like ?", param.Name+"%", param.Name+"%")
	}

	// 使用新的 Session 进行 count，避免 Distinct 影响后续查询
	err = db.Session(&gorm.Session{}).Distinct("tag_group.id").Count(&total).Error
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	param.Sorter.SetDefault()
	param.Pager.SetDefault()

	groups = make([]*TagGroup, 0)
	// PostgreSQL: 使用子查询去重，避免 DISTINCT ON 与 ORDER BY 冲突
	err = db.Preload("Tags", func(db *gorm.DB) *gorm.DB {
		return db.Order("tag.order DESC")
	}).Select("tag_group.*").
		Group("tag_group.id, tag_group.ext_corp_id, tag_group.ext_id, tag_group.name, tag_group.department_list, tag_group.\"order\", tag_group.created_at, tag_group.updated_at, tag_group.deleted_at").
		Order("tag_group.order desc, tag_group.updated_at desc").
		Offset(param.Pager.GetOffset()).Limit(param.Pager.GetLimit()).
		Find(&groups).Error
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	return groups, total, nil
}

func (tg TagGroup) Get(ExtTagGroupID string) (TagGroup, error) {
	group := TagGroup{}
	err := DB.Model(&TagGroup{}).Preload("Tags").
		Where("ext_id = ?", ExtTagGroupID).First(&group).Error
	return group, err
}

func (tg TagGroup) Create(t *TagGroup) (*TagGroup, error) {
	if len(t.DepartmentList) == 0 {
		t.DepartmentList = []int64{0}
	}

	return t, DB.Model(&TagGroup{}).Create(t).Error
}

func (tg TagGroup) Delete(extGroupIDs []string) (int64, error) {
	res := DB.Model(&TagGroup{}).Where("ext_id in ?", extGroupIDs).Delete(&TagGroup{})
	return res.RowsAffected, res.Error
}

// Upsert 在首次启动或者接收callback同步标签
// 更新标签的情况：
// 1、组内新建标签
// 2、组内删除已有某个标签 , 标签id记录在customer_staff关系中，删除标签时需要更新该处
// 3、删除标签组, 标签id记录在customer_staff关系中，删除标签时需要更新该处
// 标签更新的callback：
// 1、新建标签/标签组 可以用upsert
// 2、更新或者删除标签/标签组，callback msg有id可以直接delete
func (tg TagGroup) Upsert(group *TagGroup) error {
	if len(group.DepartmentList) == 0 {
		group.DepartmentList = []int64{0}
	}

	err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ext_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"create_time", "name", "order"})},
	).Omit("Tags").Create(&group).Error
	if err != nil {
		return err
	}

	if len(group.Tags) > 0 {
		err = DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "ext_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"create_time", "name", "order", "ext_group_id", "group_name"})},
		).Create(&(group.Tags)).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// Update
// Description: 进更新标签组内容
func (tg TagGroup) Update(group *TagGroup) error {
	if len(group.DepartmentList) == 0 {
		group.DepartmentList = []int64{0}
	}
	err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ext_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"create_time", "name", "order", "department_list"})},
	).Omit("Tags").Create(&group).Error
	if err != nil {
		return err
	}
	return nil
}

type TagGroupSwagger struct {
	List  []*TagGroup
	Pager app.Pager
}

func (tg TagGroup) TableName() string {
	return "tag_group"
}
