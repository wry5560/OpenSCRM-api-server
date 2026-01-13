package controller

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/encoding/gurl"
	"github.com/gogf/gf/util/grand"
	"github.com/pkg/errors"
	"openscrm/app/constants"
	"openscrm/app/entities"
	"openscrm/app/models"
	"openscrm/app/services"
	"openscrm/common/app"
	"openscrm/common/ecode"
	"openscrm/common/util"
	"openscrm/conf"
)

type Login struct {
	Base
	srv *services.Login
}

func NewLogin() *Login {
	return &Login{srv: services.NewDefaultLogin()}
}

// StaffAdminLogin
// @tags 企业管理
// @Summary 企业员工后台登录
// @Description 同时支持get和post请求，get可自动跳转，post可获取详细信息
// @Produce  json
// @Accept json
// @Param params body entities.StaffAdminLoginReq true "企业普通管理员登录请求"
// @Success 200 {object} app.JSONResult{data=entities.StaffAdminLoginResp} "成功"
// @Failure 400 {object} app.JSONResult{} "非法请求"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/staff_admin/action/login [get]
func (o *Login) StaffAdminLogin(c *gin.Context) {
	req := entities.StaffAdminLoginReq{}
	handler := app.NewHandler(c)
	ok, err := handler.BindAndValidateReq(&req)
	if !ok {
		handler.ResponseBadRequestError(errors.WithStack(err))
		return
	}

	if req.ExtCorpID == "" {
		req.ExtCorpID = conf.Settings.WeWork.ExtCorpID
	}

	state := grand.Letters(10)
	item, err := o.srv.StaffAdminLogin(req.ExtCorpID, state, req.SourceURL)
	if err != nil {
		err = errors.Wrap(err, "StaffAdminLogin failed")
		handler.ResponseError(err)
		return
	}

	handler.StaffAdminSession.Set(string(constants.QrcodeAuthState), state)
	// 将 source_url 存入 session，用于回调时重定向
	handler.StaffAdminSession.Set(string(constants.LoginSourceURL), req.SourceURL)
	err = handler.StaffAdminSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}
	log.Printf("[Login] State saved to session: %s", state)

	if handler.Ctx.Request.Method == "GET" {
		handler.Ctx.Redirect(http.StatusFound, item.LocationURL)
		return
	}

	handler.ResponseItem(item)
}

// StaffAdminLoginCallback
// @tags 企业管理
// @Summary 企业微信扫码登录回调
// @Produce  json
// @Accept json
// @Param params body entities.StaffAdminLoginCallbackReq true "企业普通管理员登录请求"
// @Success 200 {object} app.JSONResult{data=models.Staff} "成功"
// @Failure 400 {object} app.JSONResult{} "非法请求"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/staff_admin/action/login_callback [post]
func (o *Login) StaffAdminLoginCallback(c *gin.Context) {
	req := entities.StaffAdminLoginCallbackReq{}
	handler := app.NewHandler(c)

	// 调试日志：记录 cookie 和原始请求参数
	log.Printf("[LoginCallback] Cookies: %v", c.Request.Cookies())
	log.Printf("[LoginCallback] Raw URL: %s", c.Request.URL.String())
	log.Printf("[LoginCallback] Raw query source_url: %s", c.Query("source_url"))

	ok, err := handler.BindAndValidateReq(&req)
	if !ok {
		handler.ResponseBadRequestError(errors.WithStack(err))
		return
	}

	// 调试日志：记录绑定后的请求参数
	log.Printf("[LoginCallback] Bound req.SourceURL: %s", req.SourceURL)
	log.Printf("[LoginCallback] Bound req.State: %s", req.State)
	log.Printf("[LoginCallback] Bound req.AppID: %s", req.AppID)

	// 验证state，避免csrf攻击
	state, ok := handler.StaffAdminSession.Get(string(constants.QrcodeAuthState)).(string)
	log.Printf("[LoginCallback] Session state: %s, found: %v", state, ok)

	if !ok {
		handler.ResponseBadRequestError(errors.New("session state not found, please try login again"))
		return
	}

	if state != req.State {
		log.Printf("[LoginCallback] State mismatch: session=%s, req=%s", state, req.State)
		err = ecode.BadRequest
		handler.ResponseError(err)
		return
	}

	item, err := o.srv.StaffAdminLoginCallback(req.AppID, req.Code)
	if err != nil {
		err = errors.Wrap(err, "StaffAdminLoginCallback failed")
		handler.ResponseError(err)
		return
	}

	handler.StaffAdminSession.Set(string(constants.StaffInfo), util.JsonEncode(item))
	err = handler.StaffAdminSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}

	// 根据 source_url 重定向，支持明道云等第三方嵌入
	// 从 session 读取 source_url（因为企微 OAuth 不会保留 redirect_uri 中的自定义参数）
	redirectURL := "/staff-admin/login-callback"
	sourceURL, _ := handler.StaffAdminSession.Get(string(constants.LoginSourceURL)).(string)
	log.Printf("[LoginCallback] Session sourceURL=%s", sourceURL)

	if sourceURL != "" {
		// URL 解码
		decodedURL, decodeErr := gurl.Decode(sourceURL)
		if decodeErr == nil && decodedURL != "" {
			sourceURL = decodedURL
		}
		log.Printf("[LoginCallback] Decoded sourceURL=%s", sourceURL)

		// 如果是明道云页面，重定向到明道云回调页面
		if strings.Contains(sourceURL, "/mingdaoyun/") {
			redirectURL = "/mingdaoyun/login-callback"
			log.Printf("[LoginCallback] Detected mingdaoyun URL, redirecting to %s", redirectURL)
		} else {
			redirectURL = sourceURL
			log.Printf("[LoginCallback] Using sourceURL as redirect: %s", redirectURL)
		}
	}

	log.Printf("[LoginCallback] Final redirect URL: %s", redirectURL)
	handler.Ctx.Redirect(http.StatusFound, redirectURL)
}

// StaffAdminLogout
// @tags 企业管理
// @Summary 企业员工后台退出登录
// @Produce  json
// @Success 200 {object} app.JSONResult{} "成功"
// @Router /api/v1/staff_admin/action/logout [post]
func (o *Login) StaffAdminLogout(c *gin.Context) {
	handler := app.NewHandler(c)

	// 清除 session
	handler.StaffAdminSession.Clear()
	err := handler.StaffAdminSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}

	handler.ResponseItem(nil)
}

// StaffAdminForceLogin
// @tags 调试接口
// @Summary 指定任意用户强制登录
// @Description 仅开发和测试环境可用
// @Produce  json
// @Accept json
// @Param params body entities.StaffAdminForceLoginReq true "指定用户强制登录请求"
// @Success 200 {object} app.JSONResult{data=models.Staff} "成功"
// @Failure 400 {object} app.JSONResult{} "非法请求"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/staff_admin/action/force_login [post]
func (o *Login) StaffAdminForceLogin(c *gin.Context) {
	req := entities.StaffAdminForceLoginReq{}
	handler := app.NewHandler(c)
	ok, err := handler.BindAndValidateReq(&req)
	if !ok {
		handler.ResponseBadRequestError(errors.WithStack(err))
		return
	}

	// 生产环境不允许使用调试登录
	if conf.Settings.App.Env == constants.PROD {
		err = errors.WithStack(ecode.ForbiddenError)
		handler.ResponseError(err)
		return
	}

	if req.ExtCorpID == "" {
		req.ExtCorpID = conf.Settings.WeWork.ExtCorpID
	}

	if req.ExtStaffID == "" && len(conf.Settings.App.SuperAdmin) > 0 {
		req.ExtStaffID = conf.Settings.App.SuperAdmin[0]
	}

	item, err := (&models.Staff{}).Get(req.ExtStaffID, "", false)
	if err != nil {
		err = errors.Wrap(err, "GetStaffByUserID failed")
		handler.ResponseError(err)
		return
	}

	handler.StaffAdminSession.Set(string(constants.StaffInfo), util.JsonEncode(item))
	err = handler.StaffAdminSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}

	handler.ResponseItem(item)
}

// StaffLogin
// @tags 员工前台
// @Summary 员工H5登录
// @Description 同时支持get和post请求，get可自动跳转，post可获取详细信息
// @Produce  json
// @Accept json
// @Param params body entities.StaffLoginReq true "员工H5登录请求"
// @Success 200 {object} app.JSONResult{data=entities.StaffLoginResp} "成功"
// @Failure 400 {object} app.JSONResult{} "非法请求"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/staff_frontend/action/login [get]
func (o *Login) StaffLogin(c *gin.Context) {
	req := entities.StaffLoginReq{}
	handler := app.NewHandler(c)
	ok, err := handler.BindAndValidateReq(&req)
	if !ok {
		handler.ResponseBadRequestError(errors.WithStack(err))
		return
	}

	if req.ExtCorpID == "" {
		req.ExtCorpID = conf.Settings.WeWork.ExtCorpID
	}

	state := grand.Letters(10)
	item, err := o.srv.StaffLogin(req.ExtCorpID, state, req.SourceURL)
	if err != nil {
		err = errors.Wrap(err, "StaffLogin failed")
		handler.ResponseError(err)
		return
	}

	handler.StaffSession.Set(string(constants.QrcodeAuthState), state)
	err = handler.StaffSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}

	if handler.Ctx.Request.Method == "GET" {
		handler.Ctx.Redirect(http.StatusFound, item.LocationURL)
		return
	}

	handler.ResponseItem(item)
}

// StaffLoginCallback
// @tags 员工前台
// @Summary 员工H5登录回调
// @Produce  json
// @Accept json
// @Param params body entities.StaffLoginCallbackReq true "员工H5登录请求"
// @Success 200 {object} app.JSONResult{data=models.Staff} "成功"
// @Failure 400 {object} app.JSONResult{} "非法请求"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/staff_frontend/action/login_callback [post]
func (o *Login) StaffLoginCallback(c *gin.Context) {
	req := entities.StaffLoginCallbackReq{}
	handler := app.NewHandler(c)
	ok, err := handler.BindAndValidateReq(&req)
	if !ok {
		handler.ResponseBadRequestError(errors.WithStack(err))
		return
	}

	//// 验证state，避免csrf攻击
	//state := handler.StaffSession.Get(string(constants.QrcodeAuthState)).(string)
	//if state != req.State {
	//	err = ecode.BadRequest
	//	handler.ResponseError(err)
	//	return
	//}

	item, err := o.srv.StaffLoginCallback(req.AppID, req.Code)
	if err != nil {
		err = errors.Wrap(err, "StaffLoginCallback failed")
		handler.ResponseError(err)
		return
	}

	handler.StaffSession.Set(string(constants.StaffInfo), util.JsonEncode(item))
	err = handler.StaffSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}

	sourceURL, err := gurl.Decode(req.SourceURL)
	if err != nil {
		err = errors.Wrap(err, "invalid source_url")
		handler.ResponseBadRequestError(err)
		return
	}

	handler.Ctx.Redirect(http.StatusFound, sourceURL)

}

// StaffForceLogin
// @tags 调试接口
// @Summary 指定任意员工侧边栏强制登录
// @Description 仅开发和测试环境可用
// @Produce  json
// @Accept json
// @Param params body entities.StaffAdminForceLoginReq true "指定用户强制登录请求"
// @Success 200 {object} app.JSONResult{data=models.Staff} "成功"
// @Failure 400 {object} app.JSONResult{} "非法请求"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/staff-frontend/action/force_login [post]
func (o *Login) StaffForceLogin(c *gin.Context) {
	req := entities.StaffForceLoginReq{}
	handler := app.NewHandler(c)
	ok, err := handler.BindAndValidateReq(&req)
	if !ok {
		handler.ResponseBadRequestError(errors.WithStack(err))
		return
	}

	// 生产环境不允许使用调试登录
	if conf.Settings.App.Env == constants.PROD {
		err = errors.WithStack(ecode.ForbiddenError)
		handler.ResponseError(err)
		return
	}

	if req.ExtCorpID == "" {
		req.ExtCorpID = conf.Settings.WeWork.ExtCorpID
	}

	item, err := (&models.Staff{}).Get(req.ExtStaffID, "", false)
	if err != nil {
		err = errors.Wrap(err, "GetStaffByUserID failed")
		handler.ResponseError(err)
		return
	}

	handler.StaffSession.Set(string(constants.StaffInfo), util.JsonEncode(item))
	err = handler.StaffSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}

	handler.ResponseItem(item)
}

//// CustomerLogin
//// @tags 客户前台
//// @Summary 客户H5登录
//// @Description 同时支持get和post请求，get可自动跳转，post可获取详细信息
//// @Produce  json
//// @Accept json
//// @Param params body entities.CustomerLoginReq true "客户H5登录请求"
//// @Success 200 {object} app.JSONResult{data=entities.CustomerLoginResp} "成功"
//// @Failure 400 {object} app.JSONResult{} "非法请求"
//// @Failure 500 {object} app.JSONResult{} "内部错误"
//// @Router /api/v1/customer_frontend/action/login [get]
//func (o *Login) CustomerLogin(c *gin.Context) {
//	req := entities.CustomerLoginReq{}
//	handler := app.NewHandler(c)
//	ok, err := handler.BindAndValidateReq(&req)
//	if !ok {
//		handler.ResponseBadRequestError(errors.WithStack(err))
//		return
//	}
//
//	state := grand.Letters(10)
//	item, err := o.srv.CustomerLogin(req.ExtCorpID, state, req.SourceURL)
//	if err != nil {
//		err = errors.Wrap(err, "CustomerLogin failed")
//		handler.ResponseError(err)
//		return
//	}
//
//	handler.CustomerSession.Set(string(constants.QrcodeAuthState), state)
//	err = handler.CustomerSession.Save()
//	if err != nil {
//		err = errors.Wrap(err, "sess.Save failed")
//		handler.ResponseError(err)
//		return
//	}
//
//	if handler.Ctx.Request.Method == "GET" {
//		handler.Ctx.Redirect(http.StatusFound, item.LocationURL)
//		return
//	}
//
//	handler.ResponseItem(item)
//}

// CustomerLoginCallback
// @tags 客户前台
// @Summary 客户H5登录回调
// @Produce  json
// @Accept json
// @Param params body entities.CustomerLoginCallbackReq true "客户H5登录请求"
// @Success 200 {object} app.JSONResult{data=models.Customer} "成功"
// @Failure 400 {object} app.JSONResult{} "非法请求"
// @Failure 500 {object} app.JSONResult{} "内部错误"
// @Router /api/v1/customer_frontend/action/login_callback [post]
func (o *Login) CustomerLoginCallback(c *gin.Context) {
	req := entities.CustomerLoginCallbackReq{}
	handler := app.NewHandler(c)
	ok, err := handler.BindAndValidateReq(&req)
	if !ok {
		handler.ResponseBadRequestError(errors.WithStack(err))
		return
	}

	//// 验证state，避免csrf攻击
	//state := handler.CustomerSession.Get(string(constants.QrcodeAuthState)).(string)
	//if state != req.State {
	//	err = ecode.BadRequest
	//	handler.ResponseError(err)
	//	return
	//}

	item, err := o.srv.CustomerLoginCallback(conf.Settings.WeWork.ExtCorpID, req.Code)
	if err != nil {
		err = errors.Wrap(err, "CustomerLoginCallback failed")
		handler.ResponseError(err)
		return
	}

	handler.CustomerSession.Set(string(constants.CustomerInfo), util.JsonEncode(item))
	err = handler.CustomerSession.Save()
	if err != nil {
		err = errors.Wrap(err, "sess.Save failed")
		handler.ResponseError(err)
		return
	}

	sourceURL, err := gurl.Decode(req.SourceURL)
	if err != nil {
		err = errors.Wrap(err, "invalid source_url")
		handler.ResponseBadRequestError(err)
		return
	}

	handler.Ctx.Redirect(http.StatusFound, sourceURL)

}
