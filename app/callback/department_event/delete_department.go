package department_event

import (
	"github.com/pkg/errors"
	"openscrm/app/models"
	"openscrm/app/services"
	gowx "openscrm/pkg/easywework"
)

// EventDeleteDepartment
// Description: 删除部门事件回调
// Detail: 删除DB, 更新tagGroup的可用部门
func EventDeleteDepartment(msg *gowx.RxMessage) error {
	if msg.MsgType != gowx.MessageTypeEvent ||
		msg.Event != gowx.EventTypeChangeContact ||
		msg.ChangeType != gowx.ChangeTypeDeleteParty {
		return errors.New("wrong handler for the callback event")
	}
	eventDeleteParty, ok := msg.EventDeleteParty()
	if !ok {
		return errors.New("msg.EventEditExternalContact failed")
	}

	// 构建部门对象用于明道云同步
	department := models.Department{
		ExtCorpID: msg.ToUserID,
		ExtID:     eventDeleteParty.GetID(),
	}

	err := models.DB.
		Where("ext_corp_id = ?", msg.ToUserID).
		Where("ext_id = ?", eventDeleteParty.GetID()).
		Delete(&models.Department{}).Error
	if err != nil {
		err = errors.WithStack(err)
		return err
	}

	// 异步同步到明道云（删除部门记录）
	syncService := services.NewMingDaoYunStaffSyncService()
	syncService.AsyncSyncDepartment(&department, "delete")

	return nil
}
