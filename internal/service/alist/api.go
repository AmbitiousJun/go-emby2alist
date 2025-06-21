package alist

import (
	"fmt"
	"log"
	"net/http"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/model"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
)

// FetchResource 请求 alist 资源 url 直链
func FetchResource(fi FetchInfo) model.HttpRes[Resource] {
	if strs.AnyEmpty(fi.Path) {
		return model.HttpRes[Resource]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}
	fi.Header = CleanHeader(fi.Header)

	if !fi.UseTranscode {
		// 请求原画资源
		res := FetchFsGet(fi.Path, fi.Header)
		if res.Code == http.StatusOK {
			if link, ok := res.Data.Attr("raw_url").String(); ok {
				return model.HttpRes[Resource]{Code: http.StatusOK, Data: Resource{Url: link}}
			}
		}
		if res.Msg == "" {
			res.Msg = fmt.Sprintf("未知异常, 原始响应: %v", jsons.NewByObj(res))
		}
		return model.HttpRes[Resource]{Code: res.Code, Msg: res.Msg}
	}

	// 转码资源请求失败后, 递归请求原画资源
	failedAndTryRaw := func(originRes model.HttpRes[*jsons.Item]) model.HttpRes[Resource] {
		if !fi.TryRawIfTranscodeFail {
			return model.HttpRes[Resource]{Code: originRes.Code, Msg: originRes.Msg}
		}
		log.Printf(colors.ToRed("请求转码资源失败, 尝试请求原画资源, 原始响应: %v"), jsons.NewByObj(originRes))
		fi.UseTranscode = false
		return FetchResource(fi)
	}

	// 请求转码资源
	res := FetchFsOther(fi.Path, fi.Header)
	if res.Code != http.StatusOK {
		return failedAndTryRaw(res)
	}

	list, ok := res.Data.Attr("video_preview_play_info").Attr("live_transcoding_task_list").Done()
	if !ok || list.Type() != jsons.JsonTypeArr {
		return failedAndTryRaw(res)
	}
	idx := list.FindIdx(func(val *jsons.Item) bool { return val.Attr("template_id").Val() == fi.Format })
	if idx == -1 {
		allFmts := list.Map(func(val *jsons.Item) any { return val.Attr("template_id").Val() })
		log.Printf(colors.ToRed("查找不到指定的格式: %s, 所有可用的格式: %v"), fi.Format, jsons.NewByArr(allFmts))
		return failedAndTryRaw(res)
	}

	link, ok := list.Idx(idx).Attr("url").String()
	if !ok {
		return failedAndTryRaw(res)
	}

	// 封装字幕链接, 封装失败也不返回失败
	subtitles := make([]SubtitleInfo, 0)
	subList, ok := res.Data.Attr("video_preview_play_info").Attr("live_transcoding_subtitle_task_list").Done()
	if ok {
		subList.RangeArr(func(_ int, value *jsons.Item) error {
			lang, _ := value.Attr("language").String()
			url, _ := value.Attr("url").String()
			subtitles = append(subtitles, SubtitleInfo{
				Lang: lang,
				Url:  url,
			})
			return nil
		})
	}

	return model.HttpRes[Resource]{Code: http.StatusOK, Data: Resource{Url: link, Subtitles: subtitles}}
}

// FetchFsList 请求 alist "/api/fs/list" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsList(path string, header http.Header) model.HttpRes[*jsons.Item] {
	if strs.AnyEmpty(path) {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}
	return Fetch("/api/fs/list", http.MethodPost, header, map[string]any{
		"refresh":  true,
		"password": "",
		"path":     path,
	})
}

// FetchFsGet 请求 alist "/api/fs/get" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsGet(path string, header http.Header) model.HttpRes[*jsons.Item] {
	if strs.AnyEmpty(path) {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	return Fetch("/api/fs/get", http.MethodPost, header, map[string]any{
		"refresh":  true,
		"password": "",
		"path":     path,
	})
}

// FetchFsOther 请求 alist "/api/fs/other" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsOther(path string, header http.Header) model.HttpRes[*jsons.Item] {
	if strs.AnyEmpty(path) {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	return Fetch("/api/fs/other", http.MethodPost, header, map[string]any{
		"method":   "video_preview",
		"password": "",
		"path":     path,
	})
}

// Fetch 请求 alist api
func Fetch(uri, method string, header http.Header, body map[string]any) model.HttpRes[*jsons.Item] {
	host := config.C.Openlist.Host
	token := config.C.Openlist.Token

	// 1 发出请求
	if header == nil {
		header = make(http.Header)
	}
	header.Set("Content-Type", "application/json;charset=utf-8")
	header.Set("Authorization", token)

	resp, err := https.Request(method, host+uri).Header(header).Body(https.MapBody(body)).Do()
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "请求发送失败: " + err.Error()}
	}
	defer resp.Body.Close()

	// 2 封装响应
	result, err := jsons.Read(resp.Body)
	if err != nil {
		return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "解析响应体失败: " + err.Error()}
	}

	if code, ok := result.Attr("code").Int(); !ok || code != http.StatusOK {
		message, _ := result.Attr("message").String()
		return model.HttpRes[*jsons.Item]{Code: code, Msg: message}
	}

	if data, ok := result.Attr("data").Done(); ok {
		return model.HttpRes[*jsons.Item]{Code: http.StatusOK, Data: data}
	}

	return model.HttpRes[*jsons.Item]{Code: http.StatusBadRequest, Msg: "未知异常, result: " + result.String()}
}
