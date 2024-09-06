package emby

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/model"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/jsons"
	"go-emby2alist/internal/util/urls"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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
	if q.Get("api_key") != "" || q.Get("X-Emby-Token") != "" {
		return
	}
	q.Set("api_key", config.C.Emby.ApiKey)
	c.Request.URL.RawQuery = q.Encode()
	c.Request.Header.Del("Authorization")
}

// Fetch 请求 emby api 接口, 使用 map 请求体
//
// 如果 uri 中不包含 token, 自动从配置中取 token 进行拼接
func Fetch(uri, method string, body map[string]interface{}) (model.HttpRes[*jsons.Item], http.Header) {
	return RawFetch(uri, method, https.MapBody(body))
}

// RawFetch 请求 emby api 接口, 使用流式请求体
//
// 如果 uri 中不包含 token, 自动从配置中取 token 进行拼接
func RawFetch(uri, method string, body io.ReadCloser) (model.HttpRes[*jsons.Item], http.Header) {
	host := config.C.Emby.Host
	token := config.C.Emby.ApiKey

	// 1 检查 uri 中是否含有 token
	u := host + uri
	if !strings.Contains(uri, "api_key") && !strings.Contains(uri, "X-Emby-Token") {
		u = urls.AppendArgs(u, "api_key", token)
	}

	// 2 构造请求头, 发出请求
	header := make(http.Header)
	header.Add("Content-Type", "application/json;charset=utf-8")

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
