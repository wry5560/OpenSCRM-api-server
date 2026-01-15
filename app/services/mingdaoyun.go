package services

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"openscrm/app/constants"
	"openscrm/common/log"
	"openscrm/common/we_work"
	"openscrm/conf"
	"openscrm/pkg/easywework"
	"strings"

	"github.com/pkg/errors"
)

// MingDaoYunService 明道云服务
type MingDaoYunService struct {
	api *MingDaoYunAPI
}

// NewMingDaoYunService 创建明道云服务实例
func NewMingDaoYunService() *MingDaoYunService {
	return &MingDaoYunService{
		api: NewMingDaoYunAPI(),
	}
}

// QRCodeResult 二维码生成结果
type QRCodeResult struct {
	// QRCode 二维码图片URL
	QRCode string `json:"qr_code"`
	// ConfigID 联系方式配置ID
	ConfigID string `json:"config_id"`
}

// GetContactWayQRCode 生成企微联系我二维码
// staffId: 企微员工ID
// userNO: 明道云记录ID（rowid），用于回调时更新对应记录
func (s *MingDaoYunService) GetContactWayQRCode(staffId, userNO string) (*QRCodeResult, error) {
	if staffId == "" {
		return nil, errors.New("staffId 不能为空")
	}

	// 获取企业微信客户端
	extCorpID := conf.Settings.WeWork.ExtCorpID
	client, err := we_work.Clients.Get(extCorpID)
	if err != nil {
		return nil, errors.Wrap(err, "获取企业微信客户端失败")
	}

	// 构建 state 参数，用于回调时识别来源
	// 企微 state 限制最多30个字符
	// 格式: mdy:{encodedUserNO}
	// 将 UUID 转为 base64 编码以缩短长度（32位hex -> 22位base64）
	// 总长度 = 4(mdy:) + 22 = 26，满足30字符限制
	encodedUserNO := encodeUserNO(userNO)
	state := constants.MingDaoYunStatePrefix + encodedUserNO

	// 创建联系我二维码（永久）
	req := workwx.AddContactWay{
		Scene:      workwx.ContactWaySceneQrcode, // 二维码场景
		Type:       workwx.ContactWayTypeSingle,  // 单人模式
		User:       []string{staffId},            // 指定员工
		State:      state,                        // 回调标识
		SkipVerify: true,                         // 无需验证直接添加
	}

	log.Sugar.Infow("创建明道云联系我二维码",
		"staffId", staffId,
		"userNO", userNO,
		"state", state,
	)

	// 调用企微API创建联系方式
	configID, err := client.Customer.AddContactWay(req)
	if err != nil {
		log.Sugar.Errorw("创建联系方式失败", "err", err, "staffId", staffId)
		return nil, errors.Wrap(err, "创建联系方式失败")
	}

	// 获取联系方式详情以获取二维码URL
	contactWay, err := client.Customer.GetContactWay(configID)
	if err != nil {
		log.Sugar.Errorw("获取联系方式详情失败", "err", err, "configID", configID)
		return nil, errors.Wrap(err, "获取联系方式详情失败")
	}

	log.Sugar.Infow("二维码生成成功",
		"configID", configID,
		"qrCode", contactWay.QrCode,
	)

	return &QRCodeResult{
		QRCode:   contactWay.QrCode,
		ConfigID: configID,
	}, nil
}

// HandleAddCustomerCallback 处理来自明道云二维码的添加客户回调
// 返回值: 是否来自明道云二维码
func (s *MingDaoYunService) HandleAddCustomerCallback(event workwx.EventAddExternalContact, extCorpID string) bool {
	state := event.GetState()
	if !strings.HasPrefix(state, constants.MingDaoYunStatePrefix) {
		return false
	}

	// 提取并解码 userNO (明道云记录ID)
	encodedUserNO := strings.TrimPrefix(state, constants.MingDaoYunStatePrefix)
	if encodedUserNO == "" {
		log.Sugar.Warnw("明道云回调 userNO 为空", "state", state)
		return true
	}

	userNO := decodeUserNO(encodedUserNO)
	if userNO == "" {
		log.Sugar.Warnw("明道云回调 userNO 解码失败", "state", state, "encodedUserNO", encodedUserNO)
		return true
	}

	extStaffID := event.GetUserID()
	extCustomerID := event.GetExternalUserID()

	log.Sugar.Infow("处理明道云添加客户回调",
		"userNO", userNO,
		"extStaffID", extStaffID,
		"extCustomerID", extCustomerID,
	)

	// 异步处理，避免阻塞主流程
	go s.updateMingDaoYunCustomer(userNO, extStaffID, extCustomerID, extCorpID)

	return true
}

// updateMingDaoYunCustomer 异步更新明道云客户信息
func (s *MingDaoYunService) updateMingDaoYunCustomer(userNO, extStaffID, extCustomerID, extCorpID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Sugar.Errorw("更新明道云客户信息异常", "panic", r, "userNO", userNO)
		}
	}()

	// 获取企业微信客户端
	client, err := we_work.Clients.Get(extCorpID)
	if err != nil {
		log.Sugar.Errorw("获取企业微信客户端失败", "err", err, "extCorpID", extCorpID)
		return
	}

	// 获取客户详情
	customerInfo, err := client.Customer.GetExternalContact(extCustomerID)
	if err != nil {
		log.Sugar.Errorw("获取客户详情失败", "err", err, "extCustomerID", extCustomerID)
		return
	}

	// 构建更新信息
	info := CustomerWeComInfo{
		WecomStaffID:        extStaffID,
		WechatName:          customerInfo.ExternalContact.Name,
		WechatGender:        genderToString(customerInfo.ExternalContact.Gender),
		WecomExternalUserid: extCustomerID,
		WechatAvatar:        customerInfo.ExternalContact.Avatar,
		WechatUnionId:       customerInfo.ExternalContact.Unionid,
	}

	// 序列化对外信息（ExternalProfile 是结构体，检查是否有内容）
	if customerInfo.ExternalContact.ExternalProfile.ExternalCorpName != "" ||
		len(customerInfo.ExternalContact.ExternalProfile.ExternalAttr) > 0 {
		profileJSON, err := json.Marshal(customerInfo.ExternalContact.ExternalProfile)
		if err == nil {
			info.WecomExternalProfile = string(profileJSON)
		}
	}

	log.Sugar.Infow("准备更新明道云客户信息",
		"userNO", userNO,
		"wechatName", info.WechatName,
		"extCustomerID", extCustomerID,
	)

	// 更新明道云记录
	if err := s.api.UpdateCustomerWeComInfo(userNO, info); err != nil {
		log.Sugar.Errorw("更新明道云客户信息失败",
			"err", err,
			"userNO", userNO,
			"extCustomerID", extCustomerID,
		)
		return
	}

	log.Sugar.Infow("明道云客户信息更新成功",
		"userNO", userNO,
		"wechatName", info.WechatName,
	)
}

// genderToString 将性别枚举转换为字符串
func genderToString(gender workwx.UserGender) string {
	switch gender {
	case workwx.UserGenderMale:
		return "1"
	case workwx.UserGenderFemale:
		return "2"
	default:
		return "0"
	}
}

// encodeUserNO 将 UUID 编码为短字符串（base64）
// 32位hex UUID -> 22位base64，满足企微 state 30字符限制
func encodeUserNO(userNO string) string {
	if userNO == "" {
		return ""
	}
	// 移除 UUID 中的横线
	hexStr := strings.ReplaceAll(userNO, "-", "")
	// 解码 hex 为字节
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		// 如果不是有效的 hex，直接返回原字符串（可能已经是短ID）
		return userNO
	}
	// 编码为 base64 URL-safe，无填充
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// decodeUserNO 将编码的字符串解码回 UUID
func decodeUserNO(encoded string) string {
	if encoded == "" {
		return ""
	}
	// 尝试 base64 解码
	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		// 如果解码失败，可能是原始格式，直接返回
		return encoded
	}
	// 转为 hex 字符串
	hexStr := hex.EncodeToString(bytes)
	// 重建 UUID 格式: 8-4-4-4-12
	if len(hexStr) == 32 {
		return hexStr[:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:]
	}
	return encoded
}
