package cache

import (
	"bytes"
	"net/http"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/jsons"
)

// RespCache 公开对外暴露的缓存接口
type RespCache interface {
	// Code 响应码
	Code() int

	// Body 克隆一个响应体, 转换为缓冲区
	Body() *bytes.Buffer

	// BodyBytes 克隆一个响应体
	BodyBytes() []byte

	// JsonBody 将响应体转化成 json 返回
	JsonBody() (*jsons.Item, error)

	// Header 获取响应头属性
	Header(key string) string

	// Headers 获取克隆响应头
	Headers() http.Header

	// Space 获取缓存空间名称
	Space() string

	// SpaceKey 获取缓存空间 key
	SpaceKey() string

	// Update 更新缓存
	//
	// code 传递零值时, 会自动忽略更新
	//
	// body 传递 nil 时, 会自动忽略更新,
	// 传递空切片时, 会认为是一个空响应体进行更新
	//
	// header 传递 nil 时, 会自动忽略更新,
	// 不为 nil 时, 缓存的响应头会被清空, 并设置为新值
	Update(code int, body []byte, header http.Header)
}
