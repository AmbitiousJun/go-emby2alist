package emby

import (
	"bytes"
	"errors"
	"fmt"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/service/alist"
	"go-emby2alist/internal/service/path"
	"go-emby2alist/internal/util/jsons"
	"go-emby2alist/internal/util/urls"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// getEmbyFileLocalPath 获取 Emby 指定资源的 Path 参数
//
// uri 中必须有 query 参数 MediaSourceId,
// 如果没有携带该参数, 可能会请求到多个资源, 默认返回第一个资源
func getEmbyFileLocalPath(playbackInfoUri string) (string, error) {
	if playbackInfoUri == "" {
		return "", errors.New("参数 playbackInfoUri 不能为空")
	}

	res, _ := Fetch(playbackInfoUri, http.MethodPost, nil)
	if res.Code != http.StatusOK {
		return "", fmt.Errorf("请求 Emby 接口异常, error: %s", res.Msg)
	}

	path, ok := res.Data.Attr("MediaSources").Idx(0).Attr("Path").String()
	if !ok {
		return "", fmt.Errorf("获取不到 Path 参数, 原始响应: %v", res.Data)
	}

	return path, nil
}

// findVideoPreviewInfos 查找 source 的所有转码资源
//
// 传递 resChan 进行异步查询, 通过监听 resChan 获取查询结果
func findVideoPreviewInfos(source *jsons.Item, originName string, resChan chan []*jsons.Item) {
	if source == nil || source.Type() != jsons.JsonTypeObj {
		resChan <- nil
		return
	}

	// 转换 alist 绝对路径
	alistPathRes := path.Emby2Alist(source.Attr("Path").Val().(string))
	var transcodingList *jsons.Item
	firstFetchSuccess := false
	if alistPathRes.Success {
		res := alist.FetchFsOther(alistPathRes.Path, nil)

		if res.Code == http.StatusOK {
			if list, ok := res.Data.Attr("video_preview_play_info").Attr("live_transcoding_task_list").Done(); ok {
				firstFetchSuccess = true
				transcodingList = list
			}
		}

		if res.Code == http.StatusForbidden {
			resChan <- nil
			return
		}
	}

	// 首次请求失败, 遍历 alist 所有根目录, 重新请求
	if !firstFetchSuccess {
		paths, err := alistPathRes.Range()
		if err != nil {
			log.Printf("转换 alist 路径异常: %v", err)
			resChan <- nil
			return
		}

		for i := 0; i < len(paths); i++ {
			res := alist.FetchFsOther(paths[i], nil)
			if res.Code == http.StatusOK {
				if list, ok := res.Data.Attr("video_preview_play_info").Attr("live_transcoding_task_list").Done(); ok {
					transcodingList = list
				}
				break
			}
		}
	}

	if transcodingList == nil ||
		transcodingList.Empty() ||
		transcodingList.Type() != jsons.JsonTypeArr {
		resChan <- nil
		return
	}

	res := make([]*jsons.Item, transcodingList.Len())
	wg := sync.WaitGroup{}
	transcodingList.RangeArr(func(idx int, transcode *jsons.Item) error {
		wg.Add(1)
		go func() {
			defer wg.Done()
			copySource := jsons.NewByVal(source.Struct())
			templateId, _ := transcode.Attr("template_id").String()
			templateWidth, _ := transcode.Attr("template_width").Int()
			templateHeight, _ := transcode.Attr("template_height").Int()
			prefix := fmt.Sprintf("%s_%dx%d", templateId, templateWidth, templateHeight)
			copySource.Attr("Name").Set(fmt.Sprintf("(%s) %s", prefix, originName))

			// 重要！！！这里的 id 必须和原本的 id 不一样, 但又要确保能够正常反推出原本的 id
			newId := fmt.Sprintf("%s_%s", source.Attr("Id").Val(), prefix)
			copySource.Attr("Id").Set(newId)
			dsu, _ := copySource.Attr("DirectStreamUrl").String()
			dsu = urls.AppendArgs(dsu, "MediaSourceId", newId)

			// 标记转码资源使用转码容器
			copySource.Put("SupportsTranscoding", jsons.NewByVal(true))
			copySource.Put("TranscodingContainer", jsons.NewByVal("ts"))
			copySource.Put("TranscodingSubProtocol", jsons.NewByVal("hls"))
			copySource.Put("TranscodingUrl", jsons.NewByVal(dsu))
			copySource.Put("SupportsDirectPlay", jsons.NewByVal(false))
			copySource.Put("SupportsDirectStream", jsons.NewByVal(false))
			copySource.DelKey("DirectStreamUrl")

			res[idx] = copySource
		}()
		return nil
	})
	wg.Wait()

	resChan <- res
}

// findMediaSourceName 查找 MediaSource 中的视频名称, 如 '1080p HEVC'
func findMediaSourceName(source *jsons.Item) string {
	if source == nil || source.Type() != jsons.JsonTypeObj {
		return ""
	}

	mediaStreams, ok := source.Attr("MediaStreams").Done()
	if !ok || mediaStreams.Type() != jsons.JsonTypeArr {
		return source.Attr("Name").Val().(string)
	}

	idx := mediaStreams.FindIdx(func(val *jsons.Item) bool {
		return val.Attr("Type").Val() == "Video"
	})
	if idx == -1 {
		return source.Attr("Name").Val().(string)
	}
	return mediaStreams.Ti().Idx(idx).Attr("DisplayTitle").Val().(string)
}

// itemIdRegex 用于匹配出请求 uri 中的 itemId
var itemIdRegex = regexp.MustCompile(`(?:/emby)?/.*/(\d+)(?:/|\?)?`)

// resolveItemInfo 解析 emby 资源 item 信息
func resolveItemInfo(c *gin.Context) (ItemInfo, error) {
	if c == nil {
		return ItemInfo{}, errors.New("参数 c 不能为空")
	}

	uri := c.Request.RequestURI
	matches := itemIdRegex.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return ItemInfo{}, fmt.Errorf("itemId 匹配失败, uri: %s", uri)
	}
	itemInfo := ItemInfo{Id: matches[1], ApiKey: c.Query("X-Emby-Token")}

	if itemInfo.ApiKey == "" {
		itemInfo.ApiKey = c.Query("api_key")
	}
	if itemInfo.ApiKey == "" {
		itemInfo.ApiKey = config.C.Emby.ApiKey
	}

	msInfo, err := resolveMediaSourceId(getRequestMediaSourceId(c))
	if err != nil {
		return ItemInfo{}, fmt.Errorf("解析 MediaSource 失败, uri: %s, err: %v", uri, err)
	}
	itemInfo.MsInfo = msInfo

	u, err := url.Parse(fmt.Sprintf("/Items/%s/PlaybackInfo", itemInfo.Id))
	if err != nil {
		return ItemInfo{}, fmt.Errorf("构建 PlaybackInfo uri 失败, err: %v", err)
	}
	q := u.Query()
	q.Set("api_key", itemInfo.ApiKey)
	if !msInfo.Empty {
		q.Set("MediaSourceId", msInfo.OriginId)
	}
	u.RawQuery = q.Encode()
	itemInfo.PlaybackInfoUri = u.String()
	itemInfo.PlaybackInfoRpUri = proxyPlaybackInfoUri(u.String())

	return itemInfo, nil
}

// getRequestMediaSourceId 尝试从请求参数或请求体中获取 MediaSourceId 信息
//
// 优先返回请求参数中的值, 如果两者都获取不到, 就返回空字符串
func getRequestMediaSourceId(c *gin.Context) string {
	if c == nil {
		return ""
	}

	// 1 从请求参数中获取
	if q := strings.TrimSpace(c.Query("MediaSourceId")); q != "" {
		return q
	}

	// 2 从请求体中获取
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	reqJson, err := jsons.New(string(bodyBytes))
	if err != nil {
		return ""
	}
	if msId, ok := reqJson.Attr("MediaSourceId").String(); ok {
		return msId
	}
	return ""
}

// resolveMediaSourceId 解析 MediaSourceId
func resolveMediaSourceId(id string) (MsInfo, error) {
	res := MsInfo{Empty: true, RawId: id}

	if id == "" {
		return res, nil
	}
	res.Empty = false

	if len(id) <= 32 {
		res.OriginId = id
		return res, nil
	}

	segments := strings.Split(id, "_")
	if len(segments) != 3 {
		return MsInfo{}, errors.New("MediaSourceId 格式错误: " + id)
	}

	res.Transcode = true
	res.OriginId = segments[0]
	res.TemplateId = segments[1]
	res.Format = segments[2]
	res.SourceNamePrefix = fmt.Sprintf("%s_%s", res.TemplateId, res.Format)
	return res, nil
}
