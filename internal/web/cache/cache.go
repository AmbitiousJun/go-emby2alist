package cache

import (
	"bytes"
	"fmt"
	"go-emby2alist/internal/util/color"
	"go-emby2alist/internal/util/encrypts"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/strs"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CacheKeyIgnoreParams 忽略的请求头或者参数
//
// 如果请求地址包含列表中的请求头或者参数, 则不参与 cacheKey 运算
var CacheKeyIgnoreParams = map[string]struct{}{
	// Fileball
	"StartTimeTicks": {}, "X-Playback-Session-Id": {},

	// Emby
	"PlaySessionId": {},

	// Common
	"Range": {}, "Host": {}, "Referrer": {}, "Connection": {},
	"Accept": {}, "Accept-Encoding": {}, "Accept-Language": {}, "Cache-Control": {},
	"Upgrade-Insecure-Requests": {}, "Referer": {}, "Origin": {},

	// StreamMusic
	"X-Streammusic-Audioid": {}, "X-Streammusic-Savepath": {},

	// IP
	"X-Forwarded-For": {}, "X-Real-IP": {}, "Forwarded": {}, "Client-IP": {},
	"True-Client-IP": {}, "CF-Connecting-IP": {}, "X-Cluster-Client-IP": {},
	"Fastly-Client-IP": {}, "X-Client-IP": {}, "X-ProxyUser-IP": {},
	"Via": {}, "Forwarded-For": {}, "X-From-Cdn": {},
}

// CacheableRouteMarker 缓存白名单
// 只有匹配上正则表达式的路由才会被缓存
func CacheableRouteMarker() gin.HandlerFunc {
	cacheablePatterns := []*regexp.Regexp{
		// PlaybackInfo
		regexp.MustCompile(`(?i)^/.*items/.*/playbackinfo/reverseproxy\??`),
		// 直链重定向
		regexp.MustCompile(`(?i)^/.*(videos|audio)/.*/(stream|universal)\??`),
		// 下载
		regexp.MustCompile(`(?i)^/.*items/.*/download`),
		// 字幕
		regexp.MustCompile(`(?i)^/.*videos/.*/subtitles`),
		// 图片缓存
		regexp.MustCompile(`(?i)^/.*(/images/|\.jpg|\.png|\.jpeg|\.webp|\.ico)`),
		// 其他静态资源
		regexp.MustCompile(`(?i)^/.*(\.html|\.css|\.js|\.woff)`),
	}

	return func(c *gin.Context) {
		for _, pattern := range cacheablePatterns {
			if pattern.MatchString(c.Request.RequestURI) {
				return
			}
		}
		c.Header(HeaderKeyExpired, "-1")
	}
}

// RequestCacher 请求缓存中间件
func RequestCacher() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1 判断请求是否需要缓存
		if c.Writer.Header().Get(HeaderKeyExpired) == "-1" {
			return
		}

		// 2 计算 cache key
		cacheKey, err := calcCacheKey(c)
		if err != nil {
			log.Printf("cache key 计算异常: %v, 跳过缓存", err)
			// 如果没有调用 Abort, Gin 会自动继续调用处理器链
			return
		}

		// 3 尝试获取缓存
		if rc, ok := getCache(cacheKey); ok {
			if https.IsRedirectCode(rc.code) {
				// 适配重定向请求
				c.Redirect(rc.code, rc.header.header.Get("Location"))
			} else {
				c.Status(rc.code)
				https.CloneHeader(c, rc.header.header)
				c.Writer.Write(rc.body)
			}
			c.Abort()
			return
		}

		// 4 使用自定义的响应器
		customWriter := &respCacheWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = customWriter

		// 5 执行请求处理器
		c.Next()

		// 6 不缓存错误请求
		if https.IsErrorResponse(c) {
			return
		}

		// 7 刷新缓存
		header := c.Writer.Header()
		respHeader := respHeader{
			expired:  header.Get(HeaderKeyExpired),
			space:    header.Get(HeaderKeySpace),
			spaceKey: header.Get(HeaderKeySpaceKey),
			header:   header.Clone(),
		}
		defer header.Del(HeaderKeyExpired)
		defer header.Del(HeaderKeySpace)
		defer header.Del(HeaderKeySpaceKey)

		go putCache(cacheKey, c, customWriter.body, respHeader)
	}
}

// Duration 将一个标准的时间转换成适用于缓存时间的字符串
func Duration(d time.Duration) string {
	expired := d.Milliseconds() + time.Now().UnixMilli()
	return fmt.Sprintf("%v", expired)
}

// calcCacheKey 计算缓存 key
//
// 计算方式: 取出 请求方法, 请求路径, 请求体, 请求头 转换成字符串之后字典排序,
// 再进行 Md5Hash
func calcCacheKey(c *gin.Context) (string, error) {
	method := c.Request.Method

	q := c.Request.URL.Query()
	for key := range CacheKeyIgnoreParams {
		q.Del(key)
	}
	c.Request.URL.RawQuery = q.Encode()
	uri := c.Request.URL.String()

	body := ""
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return "", fmt.Errorf("读取请求体失败: %v", err)
		}
		body = string(bodyBytes)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	header := strings.Builder{}
	for key, values := range c.Request.Header {
		if _, ok := CacheKeyIgnoreParams[key]; ok {
			continue
		}
		header.WriteString(key)
		header.WriteString("=")
		header.WriteString(strings.Join(values, "|"))
		header.WriteString(";")
	}

	headerStr := header.String()
	preEnc := strs.Sort(c.Request.URL.RawQuery + body + headerStr)
	if headerStr != "" {
		log.Println("headers to encode cacheKey: ", color.ToYellow(headerStr))
	}

	// 为防止字典排序后, 不同的 uri 冲突, 这里在排序完的字符串前再加上原始的 uri
	uriNoArgs := strings.ReplaceAll(uri, "?"+c.Request.URL.RawQuery, "")
	uriNoArgs = strings.ReplaceAll(uriNoArgs, c.Request.URL.RawQuery, "")

	hash := encrypts.Md5Hash(method + uriNoArgs + preEnc)
	return hash, nil
}
