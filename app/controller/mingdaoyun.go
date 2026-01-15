package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"openscrm/app/services"
	"openscrm/common/app"
)

// MingDaoYunHandler 明道云接口处理器
type MingDaoYunHandler struct {
	srv *services.MingDaoYunService
}

// NewMingDaoYunHandler 创建明道云处理器实例
func NewMingDaoYunHandler() *MingDaoYunHandler {
	return &MingDaoYunHandler{
		srv: services.NewMingDaoYunService(),
	}
}

// GetQRCodeResponse 二维码响应结构
type GetQRCodeResponse struct {
	// QRCode 二维码图片URL
	QRCode string `json:"qr_code"`
	// ConfigID 联系方式配置ID（可用于后续删除）
	ConfigID string `json:"config_id"`
}

// GetQRCode 获取企微联系我二维码
// @tags 明道云
// @Summary 获取企微联系我二维码
// @Description 根据员工ID生成企业微信联系我二维码，用于明道云客户表嵌入
// @Produce json
// @Param staffId query string true "企微员工ID"
// @Param userNO query string false "明道云记录ID（rowid），用于回调时更新对应记录"
// @Success 200 {object} app.JSONResult{data=GetQRCodeResponse} "成功"
// @Failure 400 {object} app.JSONResult{} "请求错误"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/mingdaoyun/qrcode [get]
func (h *MingDaoYunHandler) GetQRCode(c *gin.Context) {
	handler := app.NewHandler(c)

	// 获取参数
	staffId := c.Query("staffId")
	userNO := c.Query("userNO")

	// 校验 staffId 必填
	if staffId == "" {
		handler.ResponseBadRequestError(errors.New("staffId 参数不能为空"))
		return
	}

	// 调用服务生成二维码
	result, err := h.srv.GetContactWayQRCode(staffId, userNO)
	if err != nil {
		handler.ResponseError(errors.Wrap(err, "生成二维码失败"))
		return
	}

	// 返回结果
	handler.ResponseItem(GetQRCodeResponse{
		QRCode:   result.QRCode,
		ConfigID: result.ConfigID,
	})
}
