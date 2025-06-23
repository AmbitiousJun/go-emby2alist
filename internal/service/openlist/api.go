package openlist

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/AmbitiousJun/go-emby2openlist/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/internal/model"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/strs"
)

// FetchResource 请求 openlist 资源 url 直链
func FetchResource(fi FetchInfo) model.HttpRes[Resource] {
	if strs.AnyEmpty(fi.Path) {
		return model.HttpRes[Resource]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}
	fi.Header = CleanHeader(fi.Header)

	if !fi.UseTranscode {
		// 请求原画资源
		res := FetchFsGet(fi.Path, fi.Header)
		if res.Code == http.StatusOK {
			return model.HttpRes[Resource]{Code: http.StatusOK, Data: Resource{Url: res.Data.RawUrl}}
		}
		return model.HttpRes[Resource]{Code: res.Code, Msg: res.Msg}
	}

	// 转码资源请求失败后, 递归请求原画资源
	failedAndTryRaw := func(originRes model.HttpRes[FsOther]) model.HttpRes[Resource] {
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

	// 匹配指定格式
	taskList := res.Data.VideoPreviewPlayInfo.LiveTranscodingTaskList
	if len(taskList) == 0 {
		return failedAndTryRaw(res)
	}
	var allFmts []string
	idx := -1
	for i, task := range taskList {
		allFmts = append(allFmts, task.TemplateId)
		if task.TemplateId == fi.Format {
			idx = i
			break
		}
	}
	if idx == -1 {
		log.Printf(colors.ToRed("查找不到指定的格式: %s, 所有可用的格式: [%s]"), fi.Format, strings.Join(allFmts, ", "))
		return failedAndTryRaw(res)
	}

	link := taskList[idx].Url
	if link == "" {
		return failedAndTryRaw(res)
	}

	return model.HttpRes[Resource]{
		Code: http.StatusOK,
		Data: Resource{
			Url:       link,
			Subtitles: res.Data.VideoPreviewPlayInfo.LiveTranscodingSubtitleTaskList,
		},
	}
}

// FetchFsList 请求 openlist "/api/fs/list" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsList(path string, header http.Header) model.HttpRes[FsList] {
	if strs.AnyEmpty(path) {
		return model.HttpRes[FsList]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	var res FsList
	err := Fetch("/api/fs/list", http.MethodPost, header, map[string]any{
		"refresh":  true,
		"password": "",
		"path":     path,
	}, &res)
	if err != nil {
		return model.HttpRes[FsList]{Code: http.StatusInternalServerError, Msg: fmt.Sprintf("FsList 请求失败: %v", err)}
	}
	return model.HttpRes[FsList]{Code: http.StatusOK, Data: res}
}

// FetchFsGet 请求 openlist "/api/fs/get" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsGet(path string, header http.Header) model.HttpRes[FsGet] {
	if strs.AnyEmpty(path) {
		return model.HttpRes[FsGet]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	var res FsGet
	err := Fetch("/api/fs/get", http.MethodPost, header, map[string]any{
		"refresh":  true,
		"password": "",
		"path":     path,
	}, &res)
	if err != nil {
		return model.HttpRes[FsGet]{Code: http.StatusInternalServerError, Msg: fmt.Sprintf("FsGet 请求失败: %v", err)}
	}
	return model.HttpRes[FsGet]{Code: http.StatusOK, Data: res}
}

// FetchFsOther 请求 openlist "/api/fs/other" 接口
//
// 传入 path 与接口的 path 作用一致
func FetchFsOther(path string, header http.Header) model.HttpRes[FsOther] {
	if strs.AnyEmpty(path) {
		return model.HttpRes[FsOther]{Code: http.StatusBadRequest, Msg: "参数 path 不能为空"}
	}

	var res FsOther
	err := Fetch("/api/fs/other", http.MethodPost, header, map[string]any{
		"method":   "video_preview",
		"password": "",
		"path":     path,
	}, &res)
	if err != nil {
		return model.HttpRes[FsOther]{Code: http.StatusInternalServerError, Msg: fmt.Sprintf("FsOther 请求失败: %v", err)}
	}
	return model.HttpRes[FsOther]{Code: http.StatusOK, Data: res}
}

// Fetch 请求 openlist api, 响应封装在 v 指针指向的结构中
func Fetch(uri, method string, header http.Header, body map[string]any, v any) error {
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
		return fmt.Errorf("Fetch 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 2 检测响应状态是否正常
	resBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Fetch 请求读取响应失败: %v", err)
	}

	var res RemoteCommonResult
	if err = json.Unmarshal(resBytes, &res); err != nil {
		return fmt.Errorf("Fetch 请求响应解析失败: %v, 响应内容: %v", err, string(resBytes))
	}
	if res.Code != http.StatusOK {
		return fmt.Errorf("Fetch 请求响应状态异常: %d, 消息: %s", res.Code, res.Message)
	}

	// 3 如果 v 参数为不为 nil 的指针, 写入响应数据
	vf := reflect.ValueOf(v)
	if vf.Kind() != reflect.Ptr || vf.IsNil() {
		return nil
	}
	if err = json.Unmarshal(res.Data, v); err != nil {
		return fmt.Errorf("Fetch 请求响应数据解析失败: %v, 响应内容: %s", err, string(res.Data))
	}
	return nil
}
