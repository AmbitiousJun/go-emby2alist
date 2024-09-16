package cache

import (
	"bytes"
	"go-emby2alist/internal/util/jsons"
	"net/http"

	"github.com/gin-gonic/gin"
)

// respCacheWriter 自定义的请求响应器
type respCacheWriter struct {
	gin.ResponseWriter               // gin 原始的响应器
	body               *bytes.Buffer // gin 回写响应时, 同步缓存
}

func (rcw *respCacheWriter) Write(b []byte) (int, error) {
	rcw.body.Write(b)
	return rcw.ResponseWriter.Write(b)
}

// respCache 存放请求的响应信息
type respCache struct {

	// code 响应码
	code int

	// body 响应体
	body []byte

	// cacheKey 缓存 key
	cacheKey string

	// expired 缓存过期时间戳 UnixMilli
	expired int64

	// header 响应头信息
	header respHeader
}

// respHeader 记录特定请求的缓存参数
type respHeader struct {
	expired  string      // 过期时间
	space    string      // 缓存空间名称
	spaceKey string      // 缓存空间 key
	header   http.Header // 原始请求的克隆请求头
}

// Code 响应码
func (c *respCache) Code() int {
	return c.code
}

// Body 克隆一个响应体, 转换为缓冲区
func (c *respCache) Body() *bytes.Buffer {
	return bytes.NewBuffer(c.BodyBytes())
}

// BodyBytes 克隆一个响应体
func (c *respCache) BodyBytes() []byte {
	return append([]byte(nil), c.body...)
}

// JsonBody 将响应体转化成 json 返回
func (c *respCache) JsonBody() (*jsons.Item, error) {
	return jsons.New(string(c.body))
}

// Header 获取响应头属性
func (c *respCache) Header(key string) string {
	return c.header.header.Get(key)
}

// Headers 获取响应头
func (c *respCache) Headers() http.Header {
	return c.header.header.Clone()
}
