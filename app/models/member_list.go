package models

import "gorm.io/gorm/clause"

type GroupChatMember struct {
	ExtCorpModel
	ExtChatID string `gorm:"index;type:char(64);uniqueIndex:idx_chat_id_user_id;comment:群聊id" json:"ext_chat_id"`
	Userid    string `gorm:"type:char(64);uniqueIndex:idx_chat_id_user_id;comment:群成员id" json:"userid"`
	// 成员类型。 1 - 企业成员 2 - 外部联系人
	Type int `gorm:"type:smallint;comment:群成员类型" json:"type"`
	// 入群时间
	JoinTime int `gorm:"type:bigint;comment:入群时间" json:"join_time"`
	//入群方式
	//1 - 由成员邀请入群（直接邀请入群）
	//2 - 由成员邀请入群（通过邀请链接入群）
	//3 - 通过扫描群二维码入群
	JoinScene int `gorm:"type:smallint;comment:入群方式" json:"join_scene"`
	// 邀请者。目前仅当是由本企业内部成员邀请入群时会返回该值
	Invitor string `gorm:"type:char(64);comment:邀请者。目前仅当是由本企业内部成员邀请入群时会返回该值" json:"invitor"`
	// 外部联系人在微信开放平台的唯一身份标识（微信unionid）
	Unionid string `gorm:"type:char(64);comment:外部联系人在微信开放平台的唯一身份标识（微信unionid）" json:"unionid"`
}

func (m GroupChatMember) TableName() string {
	return "group_chat_member"
}

func (m GroupChatMember) Upsert(list []GroupChatMember) error {
	// PostgreSQL: 使用复合唯一索引的两个列作为冲突检测
	// 同时去重避免 ON CONFLICT 报错
	uniqueMap := make(map[string]GroupChatMember)
	for _, item := range list {
		key := item.ExtChatID + "_" + item.Userid
		uniqueMap[key] = item
	}
	uniqueList := make([]GroupChatMember, 0, len(uniqueMap))
	for _, item := range uniqueMap {
		uniqueList = append(uniqueList, item)
	}

	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ext_chat_id"}, {Name: "userid"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "join_time", "join_scene", "invitor", "unionid"}),
	}).CreateInBatches(&uniqueList, len(uniqueList)).Error
}

// Delete
// Description: 根据外部客户ID删除群聊成员
// Detail: 回调时只有外部ID,则用外部ID 来删除.
func (m GroupChatMember) Delete(extCorpID string, extChatID string, userIDs []string) error {
	return DB.Where("ext_corp_id = ? and ext_chat_id = ?", extCorpID, extChatID).Where("userid not in (?)", userIDs).Delete(&GroupChatMember{}).Error
}
