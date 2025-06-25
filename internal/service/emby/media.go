package emby

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/AmbitiousJun/go-emby2openlist/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/internal/service/openlist"
	"github.com/AmbitiousJun/go-emby2openlist/internal/service/path"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/randoms"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/strs"
	"github.com/AmbitiousJun/go-emby2openlist/internal/util/urls"

	"github.com/gin-gonic/gin"
)

// MediaSourceIdSegment 自定义 MediaSourceId 的分隔符
const MediaSourceIdSegment = "[[_]]"

// getEmbyFileLocalPath 获取 Emby 指定资源的 Path 参数
//
// 优先从缓存空间中获取 PlaybackInfo 数据
//
// uri 中必须有 query 参数 MediaSourceId,
// 如果没有携带该参数, 可能会请求到多个资源, 默认返回第一个资源
func getEmbyFileLocalPath(itemInfo ItemInfo) (string, error) {
	var header http.Header
	if itemInfo.ApiKeyType == Header {
		// 带上请求头的 api key
		header = http.Header{itemInfo.ApiKeyName: []string{itemInfo.ApiKey}}
	}

	resp, err := https.Post(config.C.Emby.Host + itemInfo.PlaybackInfoUri).Header(header).Do()
	if err != nil {
		return "", fmt.Errorf("请求 Emby 接口异常, error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("请求 Emby 接口异常, error: %s", resp.Status)
	}

	type MediaSourcesHolder struct {
		MediaSources []struct {
			Path string
			Id   string
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Emby 响应异常, error: %v", err)
	}
	var holder MediaSourcesHolder
	if err = json.Unmarshal(bodyBytes, &holder); err != nil {
		return "", fmt.Errorf("解析 Emby 响应异常, error: %v, 原始响应: %s", err, string(bodyBytes))
	}

	if len(holder.MediaSources) == 0 {
		return "", fmt.Errorf("获取不到 MediaSources, 原始响应: %v", string(bodyBytes))
	}

	var path string
	var defaultPath string

	reqId := itemInfo.MsInfo.OriginId
	// 获取指定 MediaSourceId 的 Path
	for _, value := range holder.MediaSources {
		if strs.AnyEmpty(defaultPath) {
			// 默认选择第一个路径
			defaultPath = value.Path
		}
		if itemInfo.MsInfo.Empty {
			// 如果没有传递 MediaSourceId, 就使用默认的 Path
			break
		}
		if value.Id == reqId {
			path = value.Path
			break
		}
	}

	if strs.AllNotEmpty(path) {
		return path, nil
	}
	if strs.AllNotEmpty(defaultPath) {
		return defaultPath, nil
	}
	return "", fmt.Errorf("获取不到 Path 参数, 原始响应: %v", string(bodyBytes))
}

// findVideoPreviewInfos 查找 source 的所有转码资源
//
// 传递 resChan 进行异步查询, 通过监听 resChan 获取查询结果
func findVideoPreviewInfos(source *jsons.Item, originName, clientApiKey string, resChan chan []*jsons.Item) {
	if source == nil || source.Type() != jsons.JsonTypeObj {
		resChan <- nil
		return
	}

	// 转换 openlist 绝对路径
	openlistPathRes := path.Emby2Openlist(source.Attr("Path").Val().(string))
	var transcodingList []openlist.TranscodingVideoInfo
	var subtitleList []openlist.TranscodingSubtitleInfo
	firstFetchSuccess := false
	if openlistPathRes.Success {
		res := openlist.FetchFsOther(openlistPathRes.Path, nil)

		if res.Code == http.StatusOK {
			firstFetchSuccess = true
			transcodingList = res.Data.VideoPreviewPlayInfo.LiveTranscodingTaskList
			subtitleList = res.Data.VideoPreviewPlayInfo.LiveTranscodingSubtitleTaskList
		}

		if res.Code == http.StatusForbidden {
			resChan <- nil
			return
		}
	}

	// 首次请求失败, 遍历 openlist 所有根目录, 重新请求
	if !firstFetchSuccess {
		paths, err := openlistPathRes.Range()
		if err != nil {
			log.Printf("转换 openlist 路径异常: %v", err)
			resChan <- nil
			return
		}

		for _, path := range paths {
			res := openlist.FetchFsOther(path, nil)
			if res.Code == http.StatusOK {
				transcodingList = res.Data.VideoPreviewPlayInfo.LiveTranscodingTaskList
				subtitleList = res.Data.VideoPreviewPlayInfo.LiveTranscodingSubtitleTaskList
				break
			}
		}
	}

	if len(transcodingList) == 0 {
		resChan <- nil
		return
	}

	res := make([]*jsons.Item, len(transcodingList))
	wg := sync.WaitGroup{}
	itemId, _ := source.Attr("ItemId").String()
	for idx, transcode := range transcodingList {
		idx, transcode := idx, transcode
		wg.Add(1)
		go func() {
			defer wg.Done()
			if config.C.VideoPreview.IsTemplateIgnore(transcode.TemplateId) {
				// 当前清晰度被忽略
				return
			}

			copySource := jsons.FromValue(source.Struct())
			format := fmt.Sprintf("%dx%d", transcode.TemplateWidth, transcode.TemplateHeight)
			copySource.Attr("Name").Set(fmt.Sprintf("(%s_%s) %s", transcode.TemplateId, format, originName))

			// 重要！！！这里的 id 必须和原本的 id 不一样, 但又要确保能够正常反推出原本的 id
			newId := fmt.Sprintf(
				"%s%s%s%s%s%s%s",
				source.Attr("Id").Val(), MediaSourceIdSegment,
				transcode.TemplateId, MediaSourceIdSegment,
				format, MediaSourceIdSegment,
				openlist.PathEncode(openlistPathRes.Path),
			)
			copySource.Attr("Id").Set(newId)

			// 设置转码代理播放链接
			tu, _ := url.Parse(strings.ReplaceAll(MasterM3U8UrlTemplate, "${itemId}", itemId))
			q := tu.Query()
			q.Set("openlist_path", openlist.PathEncode(openlistPathRes.Path))
			q.Set("template_id", transcode.TemplateId)
			q.Set(QueryApiKeyName, clientApiKey)
			tu.RawQuery = q.Encode()

			// 标记转码资源使用转码容器
			copySource.Put("SupportsTranscoding", jsons.FromValue(true))
			copySource.Put("TranscodingContainer", jsons.FromValue("ts"))
			copySource.Put("TranscodingSubProtocol", jsons.FromValue("hls"))
			copySource.Put("TranscodingUrl", jsons.FromValue(tu.String()))
			copySource.DelKey("DirectStreamUrl")
			copySource.Put("SupportsDirectPlay", jsons.FromValue(false))
			copySource.Put("SupportsDirectStream", jsons.FromValue(false))

			// 设置转码字幕
			addSubtitles2MediaStreams(copySource, subtitleList, openlistPathRes.Path, transcode.TemplateId, clientApiKey)

			res[idx] = copySource
		}()
	}
	wg.Wait()

	// 移除 res 中的空值项
	nonNil := res[:0]
	for _, v := range res {
		if v == nil {
			continue
		}
		nonNil = append(nonNil, v)
	}
	resChan <- nonNil
}

// addSubtitles2MediaStreams 添加转码字幕到 PlaybackInfo 的 MediaStreams 项中
//
// subtitleList 是请求 openlist 转码信息接口获取到的字幕列表
func addSubtitles2MediaStreams(source *jsons.Item, subtitleList []openlist.TranscodingSubtitleInfo, openlistPath, templateId, clientApiKey string) {
	// 1 json 参数类型校验
	if source == nil || len(subtitleList) == 0 {
		return
	}
	mediaStreams, ok := source.Attr("MediaStreams").Done()
	if !ok || mediaStreams.Type() != jsons.JsonTypeArr {
		return
	}

	// 2 去除原始的字幕信息
	mediaStreams = mediaStreams.Filter(func(val *jsons.Item) bool {
		return val != nil && val.Attr("Type").Val() != "Subtitle"
	})
	source.Put("MediaStreams", mediaStreams)

	// 3 生成 MediaStream
	itemId, _ := source.Attr("ItemId").String()
	curMediaStreamsSize := mediaStreams.Len()
	fakeId := randoms.RandomHex(32)
	for index, sub := range subtitleList {
		subStream, _ := jsons.New(`{"AttachmentSize":0,"Codec":"vtt","DeliveryMethod":"External","DeliveryUrl":"/Videos/6066/4ce9f37fe8567a3898e66517b92cf2af/Subtitles/14/0/Stream.vtt?api_key=964a56845f6a4c4a8ba42204ec6f775c","DisplayTitle":"(VTT)","ExtendedVideoSubType":"None","ExtendedVideoSubTypeDescription":"None","ExtendedVideoType":"None","Index":14,"IsDefault":false,"IsExternal":true,"IsExternalUrl":false,"IsForced":false,"IsHearingImpaired":false,"IsInterlaced":false,"IsTextSubtitleStream":true,"Protocol":"File","SupportsExternalStream":true,"Type":"Subtitle"}`)

		lang := jsons.FromValue(sub.Lang)
		subStream.Put("DisplayLanguage", lang)
		subStream.Put("Language", lang)

		subName := urls.ResolveResourceName(sub.Url)
		subStream.Put("DisplayTitle", jsons.FromValue(openlist.SubLangDisplayName(sub.Lang)))
		subStream.Put("Title", jsons.FromValue(fmt.Sprintf("(%s) %s", sub.Lang, subName)))

		idx := curMediaStreamsSize + index
		subStream.Put("Index", jsons.FromValue(idx))

		u, _ := url.Parse(fmt.Sprintf("/Videos/%s/%s/Subtitles/%d/0/Stream.vtt", itemId, fakeId, idx))
		q := u.Query()
		q.Set("openlist_path", openlist.PathEncode(openlistPath))
		q.Set("template_id", templateId)
		q.Set("sub_name", subName)
		q.Set(QueryApiKeyName, clientApiKey)
		u.RawQuery = q.Encode()
		subStream.Put("DeliveryUrl", jsons.FromValue(u.String()))

		mediaStreams.Append(subStream)
	}
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

	// 匹配 item id
	uri := c.Request.RequestURI
	matches := itemIdRegex.FindStringSubmatch(uri)
	if len(matches) < 2 {
		return ItemInfo{}, fmt.Errorf("itemId 匹配失败, uri: %s", uri)
	}
	itemInfo := ItemInfo{Id: matches[1]}

	// 获取客户端请求的 api_key
	itemInfo.ApiKeyType, itemInfo.ApiKeyName, itemInfo.ApiKey = getApiKey(c)

	// 解析请求的媒体信息
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
	// 默认只携带 query 形式的 api key
	if itemInfo.ApiKeyType == Query {
		q.Set(itemInfo.ApiKeyName, itemInfo.ApiKey)
	}
	q.Set("reqformat", "json")
	q.Set("IsPlayback", "false")
	q.Set("AutoOpenLiveStream", "false")
	if !msInfo.Empty {
		q.Set("MediaSourceId", msInfo.OriginId)
	}
	u.RawQuery = q.Encode()
	itemInfo.PlaybackInfoUri = u.String()

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
	q := c.Query("MediaSourceId")
	if strs.AllNotEmpty(q) {
		return q
	}

	// 2 从请求体中获取
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var BodyHolder struct {
		MediaSourceId string
	}
	json.Unmarshal(bodyBytes, &BodyHolder)
	return BodyHolder.MediaSourceId
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

	segments := strings.Split(id, MediaSourceIdSegment)

	if len(segments) == 2 {
		res.Transcode = true
		res.OriginId = segments[0]
		res.TemplateId = segments[1]
		return res, nil
	}

	if len(segments) == 4 {
		res.Transcode = true
		res.OriginId = segments[0]
		res.TemplateId = segments[1]
		res.Format = segments[2]
		res.OpenlistPath = segments[3]
		res.SourceNamePrefix = fmt.Sprintf("%s_%s", res.TemplateId, res.Format)
		return res, nil
	}

	return MsInfo{}, errors.New("MediaSourceId 格式错误: " + id)
}

// getAllPreviewTemplateIds 获取所有转码格式
//
// 在配置文件中忽略的格式不会返回
func getAllPreviewTemplateIds() []string {
	allIds := []string{"LD", "SD", "HD", "FHD", "QHD"}

	res := []string{}
	for _, id := range allIds {
		if config.C.VideoPreview.IsTemplateIgnore(id) {
			continue
		}
		res = append(res, id)
	}
	return res
}
