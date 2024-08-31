package emby

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/web/cache"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// NoRedirectClients 不使用重定向的客户端
var NoRedirectClients = map[string]struct{}{
	"Emby for iOS":     {},
	"Emby for macOS":   {},
	"Emby for Android": {},
}

func ProxySocket() func(*gin.Context) {

	var proxy *httputil.ReverseProxy
	var once = sync.Once{}

	initFunc := func() {
		origin := config.C.Emby.Host
		u, err := url.Parse(origin)
		if err != nil {
			panic("转换 emby host 异常: " + err.Error())
		}

		proxy = httputil.NewSingleHostReverseProxy(u)

		proxy.Director = func(r *http.Request) {
			r.URL.Scheme = u.Scheme
			r.URL.Host = u.Host
		}
	}

	return func(c *gin.Context) {
		once.Do(initFunc)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// ProxySubtitles 字幕代理, 过期时间设置为 30 天
func ProxySubtitles(c *gin.Context) {
	if c == nil {
		return
	}
	c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*24*30))
	ProxyOrigin(c)
}

// ProxyOrigin 将请求代理到源服务器
func ProxyOrigin(c *gin.Context) {
	if c == nil {
		return
	}

	AddDefaultApiKey(c)
	origin := config.C.Emby.Host
	if err := https.ProxyRequest(c, origin, true); err != nil {
		c.String(http.StatusBadRequest, "代理异常: %v", err)
	}
}

// RedirectOrigin 将 GET 请求 301 重定向到源服务器
// 其他请求走本地代理
func RedirectOrigin(c *gin.Context) {
	if c == nil {
		return
	}

	if c.Request.Method != http.MethodGet {
		ProxyOrigin(c)
		return
	}

	if _, ok := NoRedirectClients[c.Query("X-Emby-Client")]; ok {
		ProxyOrigin(c)
		return
	}

	origin := config.C.Emby.Host
	c.Redirect(http.StatusMovedPermanently, origin+c.Request.URL.String())
}
