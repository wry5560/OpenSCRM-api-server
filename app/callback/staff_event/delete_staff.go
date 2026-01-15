package staff_event

import (
	"github.com/pkg/errors"
	"openscrm/app/models"
	"openscrm/app/services"
	"openscrm/common/log"
	"openscrm/conf"
	gowx "openscrm/pkg/easywework"
)

func EventDelStaffHandler(msg *gowx.RxMessage) error {
	if msg.MsgType != gowx.MessageTypeEvent ||
		msg.Event != gowx.EventTypeChangeContact ||
		msg.ChangeType != gowx.ChangeTypeDelUser {
		err := errors.New("wrong handler for the callback event")
		log.Sugar.Error("err", err)
		return err
	}

	extras, ok := msg.EventDeleteUser()
	if !ok {
		err := errors.New("msg.EventDeleteUser failed")
		log.Sugar.Errorw("get event msg failed", "err", err)
		return err
	}
	extStaffID := extras.GetUserID()
	extCorpID := conf.Settings.WeWork.ExtCorpID
	staff, err := (&models.Staff{}).Get(extStaffID, extCorpID, false)
	if err != nil {
		return err
	}

	err = models.Department{}.AddStaffNum(-1, extCorpID, staff.DeptIds)
	if err != nil {
		return err
	}

	// 异步同步员工到明道云（将状态改为离职）
	syncService := services.NewMingDaoYunStaffSyncService()
	syncService.AsyncSyncStaff(&staff, "delete")

	return nil
}
