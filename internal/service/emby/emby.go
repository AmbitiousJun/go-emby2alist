package emby

import (
	"bytes"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/color"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/jsons"
	"go-emby2alist/internal/web/cache"
	"go-emby2alist/internal/web/webport"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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

// TestProxyUri 用于测试的代理,
// 主要是为了查看实际请求的详细信息, 方便测试
func TestProxyUri(c *gin.Context) bool {
	testUris := []string{}

	flag := false
	for _, uri := range testUris {
		if strings.Contains(c.Request.RequestURI, uri) {
			flag = true
			break
		}
	}
	if !flag {
		return false
	}

	type TestInfos struct {
		Uri        string
		Method     string
		Header     map[string]string
		Body       string
		RespStatus int
		RespHeader map[string]string
		RespBody   string
	}

	infos := &TestInfos{
		Uri:        c.Request.URL.String(),
		Method:     c.Request.Method,
		Header:     make(map[string]string),
		RespHeader: make(map[string]string),
	}

	for key, values := range c.Request.Header {
		infos.Header[key] = strings.Join(values, "|")
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf(color.ToRed("测试 uri 执行异常: %v"), err)
		return false
	}
	infos.Body = string(bodyBytes)

	origin := config.C.Emby.Host
	resp, err := https.Request(infos.Method, origin+infos.Uri, c.Request.Header, io.NopCloser(bytes.NewBuffer(bodyBytes)))
	if err != nil {
		log.Printf(color.ToRed("测试 uri 执行异常: %v"), err)
		return false
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		infos.RespHeader[key] = strings.Join(values, "|")
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf(color.ToRed("测试 uri 执行异常: %v"), err)
		return false
	}
	infos.RespBody = string(bodyBytes)
	infos.RespStatus = resp.StatusCode
	log.Printf(color.ToYellow("测试 uri 代理信息: %s"), jsons.NewByVal(infos))

	c.Status(infos.RespStatus)
	c.Writer.Write(bodyBytes)

	return true
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

	port, exist := c.Get("port")
	if config.C.Ssl.Enable && (exist && port == webport.HTTPS) {
		// https 只能走代理
		ProxyOrigin(c)
		return
	}

	origin := config.C.Emby.Host
	c.Redirect(http.StatusPermanentRedirect, origin+c.Request.URL.String())
}
