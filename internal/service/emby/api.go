package emby

import (
	"errors"
	"io"
	"net/http"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/model"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/jsons"

	"github.com/gin-gonic/gin"
)

// proxyAndSetRespHeader 代理 emby 接口
// 返回响应内容, 并将响应头写入 c
//
// 如果请求是失败的响应, 会直接返回客户端, 并在第二个参数中返回 false
func proxyAndSetRespHeader(c *gin.Context) (model.HttpRes[*jsons.Item], bool) {
	c.Request.Header.Del("Accept-Encoding")
	res, respHeader := RawFetch(c.Request.URL.String(), c.Request.Method, c.Request.Header, c.Request.Body)
	if res.Code != http.StatusOK {
		checkErr(c, errors.New(res.Msg))
		return res, false
	}
	https.CloneHeader(c, respHeader)
	return res, true
}

// Fetch 请求 emby api 接口, 使用 map 请求体
func Fetch(uri, method string, header http.Header, body map[string]any) (model.HttpRes[*jsons.Item], http.Header) {
	return RawFetch(uri, method, header, https.MapBody(body))
}

// RawFetch 请求 emby api 接口, 使用流式请求体
func RawFetch(uri, method string, header http.Header, body io.ReadCloser) (model.HttpRes[*jsons.Item], http.Header) {
	u := config.C.Emby.Host + uri

	// 构造请求头, 发出请求
	if header == nil {
		header = make(http.Header)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json;charset=utf-8")
	}

	resp, err := https.Request(method, u).Header(header).Body(body).Do()
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "请求发送失败: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	// 读取响应
	result, err := jsons.Read(resp.Body)
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "解析响应失败: " + err.Error()}, nil
	}
	return model.HttpRes[*jsons.Item]{Code: http.StatusOK, Data: result}, resp.Header
}
