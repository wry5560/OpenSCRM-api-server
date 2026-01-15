package tasks

import (
	"context"
	"openscrm/app/models"
	"openscrm/app/services"
	"openscrm/common/log"
	"openscrm/common/redis"
	"openscrm/conf"
	"strconv"
	"time"
)

// MingDaoYunSync 明道云增量同步任务
type MingDaoYunSync struct {
	Base
}

// IncrementalSync 增量同步员工和部门到明道云
func (o MingDaoYunSync) IncrementalSync() {
	taskKey := "MingDaoYunIncrementalSync"

	// 获取分布式锁
	ok, err := o.Lock(taskKey, 5*time.Minute)
	if err != nil {
		log.Sugar.Errorw("获取分布式锁失败", "err", err)
		return
	}
	if !ok {
		log.Sugar.Debugw("未获取到锁，跳过本次执行")
		return
	}
	defer o.Unlock(taskKey)

	// 检查同步是否启用
	syncService := services.NewMingDaoYunStaffSyncService()
	if !syncService.IsEnabled() {
		log.Sugar.Debugw("明道云员工同步未启用，跳过")
		return
	}

	extCorpID := conf.Settings.WeWork.ExtCorpID
	taskStartTime := time.Now()

	log.Sugar.Infow("开始明道云增量同步任务", "extCorpID", extCorpID)

	// 获取上次同步时间
	lastSyncTime := o.getLastSyncTime(extCorpID)
	log.Sugar.Infow("上次同步时间", "lastSyncTime", lastSyncTime)

	// 同步部门（先部门后员工，因为员工关联部门）
	deptSuccess, deptFail := o.syncIncrementalDepartments(extCorpID, lastSyncTime, syncService)

	// 同步员工
	staffSuccess, staffFail := o.syncIncrementalStaff(extCorpID, lastSyncTime, syncService)

	// 更新上次同步时间（使用任务开始时间，避免遗漏任务执行期间的更新）
	o.setLastSyncTime(extCorpID, taskStartTime)

	log.Sugar.Infow("明道云增量同步任务完成",
		"extCorpID", extCorpID,
		"deptSuccess", deptSuccess,
		"deptFail", deptFail,
		"staffSuccess", staffSuccess,
		"staffFail", staffFail,
		"duration", time.Since(taskStartTime),
	)
}

// getLastSyncTime 获取上次同步时间
func (o MingDaoYunSync) getLastSyncTime(extCorpID string) time.Time {
	key := "mingdaoyun_sync:last_time:" + extCorpID
	val, err := redis.RedisClient.Get(context.Background(), key).Result()
	if err != nil {
		// 首次执行，返回10分钟前
		return time.Now().Add(-10 * time.Minute)
	}
	timestamp, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Now().Add(-10 * time.Minute)
	}
	return time.Unix(timestamp, 0)
}

// setLastSyncTime 设置上次同步时间
func (o MingDaoYunSync) setLastSyncTime(extCorpID string, t time.Time) {
	key := "mingdaoyun_sync:last_time:" + extCorpID
	err := redis.RedisClient.Set(context.Background(), key, t.Unix(), 0).Err()
	if err != nil {
		log.Sugar.Errorw("设置上次同步时间失败", "err", err)
	}
}

// syncIncrementalDepartments 增量同步部门
func (o MingDaoYunSync) syncIncrementalDepartments(extCorpID string, since time.Time, syncService *services.MingDaoYunStaffSyncService) (success, fail int) {
	var departments []models.Department
	err := models.DB.
		Where("ext_corp_id = ?", extCorpID).
		Where("updated_at > ?", since).
		Find(&departments).Error
	if err != nil {
		log.Sugar.Errorw("查询增量部门失败", "err", err)
		return 0, 0
	}

	if len(departments) == 0 {
		log.Sugar.Debugw("没有需要同步的部门更新")
		return 0, 0
	}

	log.Sugar.Infow("查询到待同步部门", "count", len(departments))

	for _, dept := range departments {
		err := syncService.SyncDepartmentToMingDaoYun(&dept, "update")
		if err != nil {
			log.Sugar.Errorw("同步部门失败", "deptExtId", dept.ExtID, "err", err)
			fail++
		} else {
			success++
		}
		time.Sleep(100 * time.Millisecond) // 避免限流
	}
	return
}

// syncIncrementalStaff 增量同步员工
func (o MingDaoYunSync) syncIncrementalStaff(extCorpID string, since time.Time, syncService *services.MingDaoYunStaffSyncService) (success, fail int) {
	var staffList []models.Staff
	err := models.DB.
		Where("ext_corp_id = ?", extCorpID).
		Where("updated_at > ?", since).
		Find(&staffList).Error
	if err != nil {
		log.Sugar.Errorw("查询增量员工失败", "err", err)
		return 0, 0
	}

	if len(staffList) == 0 {
		log.Sugar.Debugw("没有需要同步的员工更新")
		return 0, 0
	}

	log.Sugar.Infow("查询到待同步员工", "count", len(staffList))

	for _, staff := range staffList {
		err := syncService.SyncStaffToMingDaoYun(&staff, "update")
		if err != nil {
			log.Sugar.Errorw("同步员工失败", "staffExtId", staff.ExtID, "err", err)
			fail++
		} else {
			success++
		}
		time.Sleep(100 * time.Millisecond) // 避免限流
	}
	return
}
