package emby

import (
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/constant"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/urls"

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
	QueryApiKeyName    = "api_key"
	QueryTokenName     = "X-Emby-Token"
	HeaderAuthName     = "Authorization"
	HeaderFullAuthName = "X-Emby-Authorization"
)

const UnauthorizedResp = "Access token is invalid or expired."

// ApiKeyChecker 对指定的 api 进行鉴权
//
// 该中间件会将客户端传递的 api_key 发送给 emby 服务器, 如果 emby 返回 401 异常
// 说明这个 api_key 是客户端伪造的, 阻断客户端的请求
func ApiKeyChecker() gin.HandlerFunc {

	patterns := []*regexp.Regexp{
		regexp.MustCompile(constant.Reg_ResourceStream),
		regexp.MustCompile(constant.Reg_PlaybackInfo),
		regexp.MustCompile(constant.Reg_ItemDownload),
		regexp.MustCompile(constant.Reg_VideoSubtitles),
		regexp.MustCompile(constant.Reg_ProxyPlaylist),
		regexp.MustCompile(constant.Reg_ProxyTs),
		regexp.MustCompile(constant.Reg_ProxySubtitle),
		regexp.MustCompile(constant.Reg_ShowEpisodes),
		regexp.MustCompile(constant.Reg_UserItems),
	}

	return func(c *gin.Context) {
		// 1 取出 api_key
		kType, kName, apiKey := getApiKey(c)

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
			u = urls.AppendArgs(u, kName, apiKey)
		} else {
			header = make(http.Header)
			header.Set(kName, apiKey)
		}
		resp, err := https.Request(http.MethodGet, u, header, nil)
		if err != nil {
			log.Printf(colors.ToRed("鉴权失败: %v"), err)
			c.Abort()
			return
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf(colors.ToRed("鉴权中间件读取源服务器响应失败: %v"), err)
			bodyBytes = []byte(UnauthorizedResp)
		}
		respBody := strings.TrimSpace(string(bodyBytes))

		// 5 判断是否被源服务器拒绝
		if resp.StatusCode == http.StatusUnauthorized && respBody == UnauthorizedResp {
			c.String(http.StatusUnauthorized, "鉴权失败")
			c.Abort()
			return
		}

		// 6 校验通过, 加入信任集合
		validApiKeys.Store(apiKey, struct{}{})
	}
}

// getApiKey 获取请求中的 api_key 信息
func getApiKey(c *gin.Context) (keyType ApiKeyType, keyName string, apiKey string) {
	if c == nil {
		return Query, "", ""
	}

	keyName = QueryApiKeyName
	keyType = Query
	apiKey = c.Query(keyName)
	if strs.AllNotEmpty(apiKey) {
		return
	}

	keyName = QueryTokenName
	apiKey = c.Query(keyName)
	if strs.AllNotEmpty(apiKey) {
		return
	}

	keyType = Header
	apiKey = c.GetHeader(keyName)
	if strs.AllNotEmpty(apiKey) {
		return
	}

	keyName = HeaderAuthName
	apiKey = c.GetHeader(keyName)
	if strs.AllNotEmpty(apiKey) {
		return
	}

	keyName = HeaderFullAuthName
	apiKey = c.GetHeader(keyName)
	if strs.AllNotEmpty(apiKey) {
		return
	}

	return
}
