package cache

import (
	"bytes"
	"net/http"
	"sync"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/jsons"

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

	// mu 读写互斥控制
	mu sync.RWMutex
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
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.code
}

// Body 克隆一个响应体, 转换为缓冲区
func (c *respCache) Body() *bytes.Buffer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return bytes.NewBuffer(c.BodyBytes())
}

// BodyBytes 克隆一个响应体
func (c *respCache) BodyBytes() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]byte(nil), c.body...)
}

// JsonBody 将响应体转化成 json 返回
func (c *respCache) JsonBody() (*jsons.Item, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return jsons.New(string(c.body))
}

// Header 获取响应头属性
func (c *respCache) Header(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.header.header.Get(key)
}

// Headers 获取克隆响应头
func (c *respCache) Headers() http.Header {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.header.header.Clone()
}

// Space 获取缓存空间名称
func (c *respCache) Space() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.header.space
}

// SpaceKey 获取缓存空间 key
func (c *respCache) SpaceKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.header.spaceKey
}

// Update 更新缓存
//
// code 传递零值时, 会自动忽略更新
//
// body 传递 nil 时, 会自动忽略更新,
// 传递空切片时, 会认为是一个空响应体进行更新
//
// header 传递 nil 时, 会自动忽略更新,
// 不为 nil 时, 缓存的响应头会被清空, 并设置为新值
func (c *respCache) Update(code int, body []byte, header http.Header) {
	if code == 0 && body == nil && header == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if code != 0 {
		c.code = code
	}

	if body != nil {
		// 新建一个底层数组来存放响应体数据
		c.body = append(([]byte)(nil), body...)
	}

	if header != nil {
		c.header.header = header.Clone()
	}
}
