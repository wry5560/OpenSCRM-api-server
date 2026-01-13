package session

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"openscrm/common/log"
)

var Store sessions.Store

func Setup(redisHost, redisPassword, aesKey string) {
	var err error
	Store, err = redis.NewStore(10, "tcp", redisHost, redisPassword, []byte(aesKey))
	if err != nil {
		log.Sugar.Fatalw("setup session failed", "err", err)
		return
	}

	// 设置 Cookie 选项，支持跨域 iframe 嵌入（明道云集成）
	// SameSite=None 允许跨站请求携带 Cookie
	// Secure=true 要求 HTTPS 环境
	Store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})
}
