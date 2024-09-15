package emby

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/constant"
	"go-emby2alist/internal/util/colors"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/strs"
	"go-emby2alist/internal/util/urls"
	"log"
	"net/http"
	"regexp"
	"sync"

	"github.com/gin-gonic/gin"
)

// AuthUri 鉴权地址
//
// 通过此 uri, 可以判断出客户端传递的 api_key 是否是被 emby 服务器认可的
const AuthUri = "/emby/Auth/Keys"

// validApiKeys 已经校验通过的 api_key, 下次就不再校验
//
// 这个 map 不会进行大小限制, 考虑到 emby 原服务器中合法的 api_key 个数不是无限个
// 所以这里也不用限制太多
var validApiKeys = sync.Map{}

// ApiKeyType 标记 emby 支持的不同种 api_key 传递方式
type ApiKeyType string

const (
	Query  ApiKeyType = "query"  // query 参数中的 api_key
	Header ApiKeyType = "header" // 请求头中的 Authorization
)

const (
	QueryApiKeyName = "api_key"
	QueryTokenName  = "X-Emby-Token"
	HeaderAuthName  = "Authorization"
)

// ApiKeyChecker 对指定的 api 进行鉴权
//
// 该中间件会将客户端传递的 api_key 发送给 emby 服务器, 如果 emby 返回 401 异常
// 说明这个 api_key 是客户端伪造的, 阻断客户端的请求
func ApiKeyChecker() gin.HandlerFunc {

	patterns := []*regexp.Regexp{
		regexp.MustCompile(constant.Reg_ResourceStream),
		regexp.MustCompile(constant.Reg_ItemDownload),
		regexp.MustCompile(constant.Reg_VideoSubtitles),
		regexp.MustCompile(constant.Reg_ProxyPlaylist),
		regexp.MustCompile(constant.Reg_ProxyTs),
		regexp.MustCompile(constant.Reg_ProxySubtitle),
	}

	return func(c *gin.Context) {
		// 1 取出 api_key
		kType, apiKey := getApiKey(c)

		// 2 如果该 key 已经是被信任的, 跳过校验
		if _, ok := validApiKeys.Load(apiKey); ok {
			return
		}

		// 3 判断当前请求的 uri 是否需要被校验
		needCheck := false
		for _, pattern := range patterns {
			if pattern.MatchString(c.Request.RequestURI) {
				needCheck = true
				break
			}
		}
		if !needCheck {
			return
		}

		// 4 发出请求, 验证 api_key
		u := config.C.Emby.Host + AuthUri
		var header http.Header
		if kType == Query {
			u = urls.AppendArgs(u, QueryApiKeyName, apiKey)
		} else {
			header = make(http.Header)
			header.Set(HeaderAuthName, apiKey)
		}
		resp, err := https.Request(http.MethodGet, u, header, nil)
		if err != nil {
			log.Printf(colors.ToRed("鉴权失败: %v"), err)
			c.Abort()
			return
		}
		defer resp.Body.Close()

		// 5 判断是否是 401 响应码
		if resp.StatusCode == http.StatusUnauthorized {
			c.String(http.StatusUnauthorized, "鉴权失败")
			c.Abort()
			return
		}

		// 6 校验通过, 加入信任集合
		validApiKeys.Store(apiKey, struct{}{})
	}
}

// getApiKey 获取请求中的 api_key 信息
func getApiKey(c *gin.Context) (ApiKeyType, string) {
	if c == nil {
		return Query, ""
	}

	apiKey := c.Query(QueryApiKeyName)
	if strs.AllNotEmpty(apiKey) {
		return Query, apiKey
	}

	apiKey = c.Query(QueryTokenName)
	if strs.AllNotEmpty(apiKey) {
		return Query, apiKey
	}

	apiKey = c.GetHeader(HeaderAuthName)
	if strs.AllNotEmpty(apiKey) {
		return Header, apiKey
	}

	return Query, ""
}
