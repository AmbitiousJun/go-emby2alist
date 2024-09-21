package emby

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/model"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/urls"

	"github.com/gin-gonic/gin"
)

// proxyAndSetRespHeader 代理 emby 接口
// 返回响应内容, 并将响应头写入 c
//
// 如果请求是失败的响应, 会直接返回客户端, 并在第二个参数中返回 false
func proxyAndSetRespHeader(c *gin.Context) (model.HttpRes[*jsons.Item], bool) {
	res, respHeader := RawFetch(c.Request.URL.String(), c.Request.Method, nil, c.Request.Body)
	if res.Code != http.StatusOK {
		checkErr(c, errors.New(res.Msg))
		return res, false
	}
	https.CloneHeader(c, respHeader)
	return res, true
}

// AddDefaultApiKey 为请求加上 api_key
//
// 如果检测到已经包含了 api_key 或者 X-Emby-Token 则取消操作
//
// 如果已经成功加上了 api_key, 则移除请求头的 Authorization 属性
func AddDefaultApiKey(c *gin.Context) {
	if c == nil {
		return
	}
	q := c.Request.URL.Query()
	if q.Get(QueryApiKeyName) != "" || q.Get(QueryTokenName) != "" {
		return
	}
	q.Set(QueryApiKeyName, config.C.Emby.ApiKey)
	c.Request.URL.RawQuery = q.Encode()
	c.Request.Header.Del("Authorization")
}

// Fetch 请求 emby api 接口, 使用 map 请求体
//
// 如果 uri 中不包含 token, 自动从配置中取 token 进行拼接
func Fetch(uri, method string, header http.Header, body map[string]interface{}) (model.HttpRes[*jsons.Item], http.Header) {
	return RawFetch(uri, method, header, https.MapBody(body))
}

// RawFetch 请求 emby api 接口, 使用流式请求体
//
// 如果 uri 中不包含 token, 自动从配置中取 token 进行拼接
func RawFetch(uri, method string, header http.Header, body io.ReadCloser) (model.HttpRes[*jsons.Item], http.Header) {
	host := config.C.Emby.Host
	token := config.C.Emby.ApiKey

	// 1 检查 uri 中是否含有 token
	u := host + uri
	if !strings.Contains(uri, QueryApiKeyName) && !strings.Contains(uri, "QueryTokenName") {
		u = urls.AppendArgs(u, QueryApiKeyName, token)
	}

	// 2 构造请求头, 发出请求
	if header == nil {
		header = make(http.Header)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json;charset=utf-8")
	}

	resp, err := https.Request(method, u, header, body)
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "请求发送失败: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	// 3 读取响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "读取响应失败: " + err.Error()}, nil
	}
	result, err := jsons.New(string(bodyBytes))
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "解析响应失败: " + err.Error()}, nil
	}
	return model.HttpRes[*jsons.Item]{Code: http.StatusOK, Data: result}, resp.Header
}
