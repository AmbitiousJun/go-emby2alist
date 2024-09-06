package alist

import (
	"fmt"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/model"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/jsons"
	"io"
	"log"
	"net/http"
	"strings"
)

// FetchResource 请求 alist 资源 url 直链
//
//	path: alist 资源绝对路径
//	useTranscode: 是否请求转码资源 (只支持视频资源, 如果该项为 false, 则后两个参数可任意传递)
//	format: 要请求的转码资源的格式, 如: FHD
//	tryRawIfTranscodeFail: 如果请求转码资源失败, 是否尝试请求原画资源
func FetchResource(path string, useTranscode bool, format string, tryRawIfTranscodeFail bool) model.HttpRes[string] {
	if path = strings.TrimSpace(path); path == "" {
		return model.HttpRes[string]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	if !useTranscode {
		// 请求原画资源
		res := FetchFsGet(path)
		if res.Code == http.StatusOK {
			if link, ok := res.Data.Attr("raw_url").String(); ok {
				return model.HttpRes[string]{Code: http.StatusOK, Data: link}
			}
		}
		if res.Msg == "" {
			res.Msg = fmt.Sprintf("未知异常, 原始响应: %v", jsons.NewByObj(res))
		}
		return model.HttpRes[string]{Code: res.Code, Msg: res.Msg}
	}

	// 转码资源请求失败后, 递归请求原画资源
	failedAndTryRaw := func(originRes model.HttpRes[*jsons.Item]) model.HttpRes[string] {
		if !tryRawIfTranscodeFail {
			return model.HttpRes[string]{Code: originRes.Code, Msg: originRes.Msg}
		}
		log.Printf("请求转码资源失败, 尝试请求原画资源, 原始响应: %v", jsons.NewByObj(originRes))
		return FetchResource(path, false, "", false)
	}

	// 请求转码资源
	res := FetchFsOther(path)
	if res.Code != http.StatusOK {
		return failedAndTryRaw(res)
	}

	list, ok := res.Data.Attr("video_preview_play_info").Attr("live_transcoding_task_list").Done()
	if !ok || list.Type() != jsons.JsonTypeArr {
		return failedAndTryRaw(res)
	}
	idx := list.FindIdx(func(val *jsons.Item) bool { return val.Attr("template_id").Val() == format })
	if idx == -1 {
		allFmts := list.Map(func(val *jsons.Item) interface{} { return val.Attr("template_id").Val() })
		log.Printf("查找不到指定的格式: %s, 所有可用的格式: %v", format, jsons.NewByArr(allFmts))
		return failedAndTryRaw(res)
	}

	link, ok := list.Idx(idx).Attr("url").String()
	if !ok {
		return failedAndTryRaw(res)
	}

	return model.HttpRes[string]{Code: http.StatusOK, Data: link}
}

// FetchFsList 请求 alist "/api/fs/list" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsList(path string) model.HttpRes[*jsons.Item] {
	if path = strings.TrimSpace(path); path == "" {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}
	return Fetch("/api/fs/list", http.MethodPost, map[string]interface{}{
		"refresh":  true,
		"password": "",
		"path":     path,
	})
}

// FetchFsGet 请求 alist "/api/fs/get" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsGet(path string) model.HttpRes[*jsons.Item] {
	if path = strings.TrimSpace(path); path == "" {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	return Fetch("/api/fs/get", http.MethodPost, map[string]interface{}{
		"refresh":  true,
		"password": "",
		"path":     path,
	})
}

// FetchFsOther 请求 alist "/api/fs/other" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsOther(path string) model.HttpRes[*jsons.Item] {
	if path = strings.TrimSpace(path); path == "" {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	return Fetch("/api/fs/other", http.MethodPost, map[string]interface{}{
		"method":   "video_preview",
		"password": "",
		"path":     path,
	})
}

// Fetch 请求 alist api
func Fetch(uri, method string, body map[string]interface{}) model.HttpRes[*jsons.Item] {
	host := config.C.Alist.Host
	token := config.C.Alist.Token

	// 1 发出请求
	header := make(http.Header)
	header.Add("Content-Type", "application/json;charset=utf-8")
	header.Add("Authorization", token)

	resp, err := https.Request(method, host+uri, header, https.MapBody(body))
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "请求发送失败: " + err.Error()}
	}
	defer resp.Body.Close()

	// 2 封装响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "读取响应体失败: " + err.Error()}
	}
	result, err := jsons.New(string(bodyBytes))
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "解析响应体失败: " + err.Error()}
	}

	if code, ok := result.Attr("code").Int(); !ok || code != http.StatusOK {
		return model.HttpRes[*jsons.Item]{Code: code, Msg: result.Attr("message").Val().(string)}
	}

	if data, ok := result.Attr("data").Done(); ok {
		return model.HttpRes[*jsons.Item]{Code: http.StatusOK, Data: data}
	}

	return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "未知异常, result: " + result.String()}
}
