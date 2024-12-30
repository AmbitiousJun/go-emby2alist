package emby

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/web/cache"

	"github.com/gin-gonic/gin"
)

const (

	// PlaybackCacheSpace PlaybackInfo 的缓存空间 key
	PlaybackCacheSpace = "PlaybackInfo"

	// MasterM3U8UrlTemplate 转码 m3u8 地址模板
	MasterM3U8UrlTemplate = `/videos/${itemId}/master.m3u8?DeviceId=a690fc29-1f3e-423b-ba23-f03049361a3b\u0026MediaSourceId=83ed6e4e3d820864a3d07d2ef9efab2e\u0026PlaySessionId=9f01e60a22c74ad0847319175912663b\u0026api_key=f53f3bf34c0543ed81415b86576058f2\u0026LiveStreamId=06044cf0e6f93cdae5f285c9ecfaaeb4_01413a525b3a9622ce6fdf19f7dde354_83ed6e4e3d820864a3d07d2ef9efab2e\u0026VideoCodec=h264,h265,hevc,av1\u0026AudioCodec=mp3,aac\u0026VideoBitrate=6808000\u0026AudioBitrate=192000\u0026AudioStreamIndex=1\u0026TranscodingMaxAudioChannels=2\u0026SegmentContainer=ts\u0026MinSegments=1\u0026BreakOnNonKeyFrames=True\u0026SubtitleStreamIndexes=-1\u0026ManifestSubtitles=vtt\u0026h264-profile=high,main,baseline,constrainedbaseline,high10\u0026h264-level=62\u0026hevc-codectag=hvc1,hev1,hevc,hdmv`

	// PlaybackCommonPayload 请求 PlaybackInfo 的通用请求体
	PlaybackCommonPayload = `{"DeviceProfile":{"MaxStaticBitrate":140000000,"MaxStreamingBitrate":140000000,"MusicStreamingTranscodingBitrate":192000,"DirectPlayProfiles":[{"Container":"mp4,m4v","Type":"Video","VideoCodec":"h264,h265,hevc,av1,vp8,vp9","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"mkv","Type":"Video","VideoCodec":"h264,h265,hevc,av1,vp8,vp9","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"flv","Type":"Video","VideoCodec":"h264","AudioCodec":"aac,mp3"},{"Container":"3gp","Type":"Video","VideoCodec":"","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"mov","Type":"Video","VideoCodec":"h264","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"opus","Type":"Audio"},{"Container":"mp3","Type":"Audio","AudioCodec":"mp3"},{"Container":"mp2,mp3","Type":"Audio","AudioCodec":"mp2"},{"Container":"m4a","AudioCodec":"aac","Type":"Audio"},{"Container":"mp4","AudioCodec":"aac","Type":"Audio"},{"Container":"flac","Type":"Audio"},{"Container":"webma,webm","Type":"Audio"},{"Container":"wav","Type":"Audio","AudioCodec":"PCM_S16LE,PCM_S24LE"},{"Container":"ogg","Type":"Audio"},{"Container":"webm","Type":"Video","AudioCodec":"vorbis,opus","VideoCodec":"av1,VP8,VP9"}],"TranscodingProfiles":[{"Container":"aac","Type":"Audio","AudioCodec":"aac","Context":"Streaming","Protocol":"hls","MaxAudioChannels":"2","MinSegments":"1","BreakOnNonKeyFrames":true},{"Container":"aac","Type":"Audio","AudioCodec":"aac","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mp3","Type":"Audio","AudioCodec":"mp3","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"opus","Type":"Audio","AudioCodec":"opus","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"wav","Type":"Audio","AudioCodec":"wav","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"opus","Type":"Audio","AudioCodec":"opus","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mp3","Type":"Audio","AudioCodec":"mp3","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"aac","Type":"Audio","AudioCodec":"aac","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"wav","Type":"Audio","AudioCodec":"wav","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mkv","Type":"Video","AudioCodec":"mp3,aac,opus,flac,vorbis","VideoCodec":"h264,h265,hevc,av1,vp8,vp9","Context":"Static","MaxAudioChannels":"2","CopyTimestamps":true},{"Container":"ts","Type":"Video","AudioCodec":"mp3,aac","VideoCodec":"h264,h265,hevc,av1","Context":"Streaming","Protocol":"hls","MaxAudioChannels":"2","MinSegments":"1","BreakOnNonKeyFrames":true,"ManifestSubtitles":"vtt"},{"Container":"webm","Type":"Video","AudioCodec":"vorbis","VideoCodec":"vpx","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mp4","Type":"Video","AudioCodec":"mp3,aac,opus,flac,vorbis","VideoCodec":"h264","Context":"Static","Protocol":"http"}],"ContainerProfiles":[],"CodecProfiles":[{"Type":"VideoAudio","Codec":"aac","Conditions":[{"Condition":"Equals","Property":"IsSecondaryAudio","Value":"false","IsRequired":"false"}]},{"Type":"VideoAudio","Conditions":[{"Condition":"Equals","Property":"IsSecondaryAudio","Value":"false","IsRequired":"false"}]},{"Type":"Video","Codec":"h264","Conditions":[{"Condition":"EqualsAny","Property":"VideoProfile","Value":"high|main|baseline|constrained baseline|high 10","IsRequired":false},{"Condition":"LessThanEqual","Property":"VideoLevel","Value":"62","IsRequired":false}]},{"Type":"Video","Codec":"hevc","Conditions":[{"Condition":"EqualsAny","Property":"VideoCodecTag","Value":"hvc1|hev1|hevc|hdmv","IsRequired":false}]}],"SubtitleProfiles":[{"Format":"vtt","Method":"Hls"},{"Format":"eia_608","Method":"VideoSideData","Protocol":"hls"},{"Format":"eia_708","Method":"VideoSideData","Protocol":"hls"},{"Format":"vtt","Method":"External"},{"Format":"ass","Method":"External"},{"Format":"ssa","Method":"External"}],"ResponseProfiles":[{"Type":"Video","Container":"m4v","MimeType":"video/mp4"}]}}`
)

var (

	// ValidCacheItemsTypeRegex 校验 Items 的 Type 参数, 通过正则才覆盖 PlaybackInfo 缓存
	ValidCacheItemsTypeRegex = regexp.MustCompile(`(?i)(movie|episode)`)

	// UnvalidCacheItemsUARegex 特定客户端的 Items 请求, 不覆盖 PlaybackInfo 缓存
	UnvalidCacheItemsUARegex = regexp.MustCompile(`(?i)(infuse)`)
)

// TransferPlaybackInfo 代理 PlaybackInfo 接口, 防止客户端转码
func TransferPlaybackInfo(c *gin.Context) {
	// 1 解析资源信息
	itemInfo, err := resolveItemInfo(c)
	log.Printf(colors.ToBlue("ItemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))
	if checkErr(c, err) {
		return
	}

	// 如果是远程资源, 直接代理到源服务器
	if handleRemotePlayback(c, itemInfo) {
		c.Header(cache.HeaderKeyExpired, "-1")
		return
	}

	// 如果是指定 MediaSourceId 的 PlaybackInfo 信息, 就从缓存空间中获取
	msInfo := itemInfo.MsInfo
	if useCacheSpacePlaybackInfo(c, itemInfo) {
		c.Header(cache.HeaderKeyExpired, "-1")
		return
	}

	// 2 请求 emby 源服务器的 PlaybackInfo 信息
	c.Request.Header.Del("Accept-Encoding")
	originRequestBody := c.Request.Body
	c.Request.Body = io.NopCloser(bytes.NewBufferString(PlaybackCommonPayload))
	res, respHeader := RawFetch(itemInfo.PlaybackInfoUri, c.Request.Method, c.Request.Header, c.Request.Body)
	if res.Code != http.StatusOK {
		checkErr(c, errors.New(res.Msg))
		return
	}

	// 3 处理 JSON 响应
	resJson := res.Data
	mediaSources, ok := resJson.Attr("MediaSources").Done()
	if !ok || mediaSources.Type() != jsons.JsonTypeArr {
		checkErr(c, errors.New("获取不到 MediaSources 属性"))
	}

	if mediaSources.Empty() {
		log.Println(colors.ToYellow("没有找到可播放的资源"))
		c.JSON(res.Code, resJson.Struct())
		return
	}

	log.Printf(colors.ToBlue("获取到的 MediaSources 个数: %d"), mediaSources.Len())
	var haveReturned = errors.New("have returned")
	resChans := make([]chan []*jsons.Item, 0)
	err = mediaSources.RangeArr(func(_ int, source *jsons.Item) error {
		if !msInfo.Empty {
			// 如果客户端请求携带了 MediaSourceId 参数
			// 在返回数据时, 需要重新设置回原始的 Id
			source.Attr("Id").Set(msInfo.RawId)
		}

		iis, _ := source.Attr("IsInfiniteStream").Bool()
		if iis {
			// 默认无限流为电视直播, 代理到源服务器
			c.Request.Body = originRequestBody
			ProxyOrigin(c)
			return haveReturned
		}

		// 转换直链链接
		source.Put("SupportsDirectPlay", jsons.NewByVal(true))
		source.Put("SupportsDirectStream", jsons.NewByVal(true))
		newUrl := fmt.Sprintf(
			"/videos/%s/stream?MediaSourceId=%s&%s=%s&Static=true",
			itemInfo.Id, source.Attr("Id").Val(), QueryApiKeyName, itemInfo.ApiKey,
		)
		source.Put("DirectStreamUrl", jsons.NewByVal(newUrl))
		log.Printf(colors.ToBlue("设置直链播放链接为: %s"), newUrl)

		// 简化资源名称
		name := findMediaSourceName(source)
		if name != "" {
			source.Put("Name", jsons.NewByVal(name))
		}
		name = source.Attr("Name").Val().(string)
		source.Attr("Name").Set(fmt.Sprintf("(原画) %s", name))

		source.Put("SupportsTranscoding", jsons.NewByVal(false))
		source.DelKey("TranscodingUrl")
		source.DelKey("TranscodingSubProtocol")
		source.DelKey("TranscodingContainer")
		log.Println(colors.ToBlue("转码配置被移除"))

		// 如果是远程资源, 不获取转码地址
		ir, _ := source.Attr("IsRemote").Bool()
		if ir {
			return nil
		}

		// 添加转码 MediaSource 获取
		cfg := config.C.VideoPreview
		if !msInfo.Empty || !cfg.Enable || !cfg.ContainerValid(source.Attr("Container").Val().(string)) {
			return nil
		}
		resChan := make(chan []*jsons.Item, 1)
		go findVideoPreviewInfos(source, name, itemInfo.ApiKey, resChan)
		resChans = append(resChans, resChan)
		return nil
	})

	if err == haveReturned {
		return
	}

	defer func() {
		// 缓存 12h
		c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*12))
		// 将请求结果缓存到指定缓存空间下
		c.Header(cache.HeaderKeySpace, PlaybackCacheSpace)
		c.Header(cache.HeaderKeySpaceKey, calcPlaybackInfoSpaceCacheKey(itemInfo))
	}()

	// 收集异步请求的转码资源信息
	for _, resChan := range resChans {
		previewInfos := <-resChan
		if len(previewInfos) > 0 {
			log.Printf(colors.ToGreen("找到 %d 个转码资源信息"), len(previewInfos))
			mediaSources.Append(previewInfos...)
		}
	}

	respHeader.Del("Content-Length")
	https.CloneHeader(c, respHeader)
	c.JSON(res.Code, resJson.Struct())
}

// handleRemotePlayback 判断如果请求的 PlaybackInfo 信息是远程地址, 直接返回结果
func handleRemotePlayback(c *gin.Context, itemInfo ItemInfo) bool {
	// 请求必须携带 MediaSourceId
	if itemInfo.MsInfo.Empty {
		return false
	}

	c.Request.Header.Del("Accept-Encoding")
	originRequestBody := c.Request.Body
	c.Request.Body = io.NopCloser(bytes.NewBufferString(PlaybackCommonPayload))
	res, _ := RawFetch(itemInfo.PlaybackInfoUri, c.Request.Method, c.Request.Header, c.Request.Body)
	if res.Code != http.StatusOK {
		return false
	}

	mediaSources, ok := res.Data.Attr("MediaSources").Done()
	if !ok || mediaSources.Len() != 1 {
		return false
	}

	ms, _ := mediaSources.Idx(0).Done()
	iis, _ := ms.Attr("IsInfiniteStream").Bool()
	if iis {
		// 默认无限流为电视直播, 直接代理到源服务器
		c.Request.Body = originRequestBody
		ProxyOrigin(c)
		return true
	}

	return false
}

// useCacheSpacePlaybackInfo 请求缓存空间的 PlaybackInfo 信息, 前提是开启了缓存功能
//
// ① 请求携带 MediaSourceId:
//
//	从缓存空间的全量缓存中匹配 MediaSourceId, 没有全量缓存或者匹配失败直接报 500 错误
//
// ② 请求不携带 MediaSourceId:
//
//	先判断缓存空间是否有缓存, 没有缓存返回 false, 由主函数请求全量信息并缓存
//	有缓存则直接返回缓存中的全量信息
func useCacheSpacePlaybackInfo(c *gin.Context, itemInfo ItemInfo) bool {
	if c == nil {
		return false
	}
	reqId, err := url.QueryUnescape(itemInfo.MsInfo.RawId)
	if err != nil {
		return false
	}

	if !config.C.Cache.Enable {
		// 未开启缓存功能
		return false
	}

	// updateCache 刷新缓存空间的缓存
	//
	// 1 将 targetIdx 的 MediaSource 移至最前
	// 2 更新所有与 target 一致 ItemId 的 DefaultAudioStreamIndex 和 DefaultSubtitleStreamIndex
	updateCache := func(spaceCache cache.RespCache, jsonBody *jsons.Item, targetIdx int) {
		// 获取所有的 MediaSources
		mediaSources, ok := jsonBody.Attr("MediaSources").Done()
		if !ok {
			return
		}

		// 获取目标 MediaSource
		targetMs, ok := mediaSources.Idx(targetIdx).Done()
		if !ok {
			return
		}

		// 更新 MediaSource
		newAdoIdx, err := strconv.Atoi(c.Query("AudioStreamIndex"))
		var newAdoVal *jsons.Item
		if err == nil {
			newAdoVal = jsons.NewByVal(newAdoIdx)
			targetMs.Put("DefaultAudioStreamIndex", newAdoVal)
		}
		var newSubVal *jsons.Item
		newSubIdx, err := strconv.Atoi(c.Query("SubtitleStreamIndex"))
		if err == nil {
			newSubVal = jsons.NewByVal(newSubIdx)
			targetMs.Put("DefaultSubtitleStreamIndex", newSubVal)
		}

		// 准备一个新的 MediaSources 数组
		newMediaSources := jsons.NewEmptyArr()
		newMediaSources.Append(targetMs)
		targetItemId, _ := targetMs.Attr("ItemId").String()
		mediaSources.RangeArr(func(index int, value *jsons.Item) error {
			if index == targetIdx {
				return nil
			}
			curItemId, _ := value.Attr("ItemId").String()
			if curItemId == targetItemId {
				if newAdoVal != nil {
					value.Put("DefaultAudioStreamIndex", newAdoVal)
				}
				if newSubVal != nil {
					value.Put("DefaultSubtitleStreamIndex", newSubVal)
				}
			}
			newMediaSources.Append(value)
			return nil
		})
		jsonBody.Put("MediaSources", newMediaSources)

		// 更新缓存
		newBody := []byte(jsonBody.String())
		newHeader := spaceCache.Headers()
		newHeader.Set("Content-Length", strconv.Itoa(len(newBody)))
		spaceCache.Update(0, newBody, newHeader)
		log.Printf(colors.ToPurple("刷新缓存空间 PlaybackInfo 信息, space: %s, spaceKey: %s"), spaceCache.Space(), spaceCache.SpaceKey())
	}

	// findMediaSourceAndReturn 从全量 PlaybackInfo 信息中查询指定 MediaSourceId 信息
	// 处理成功返回 true
	findMediaSourceAndReturn := func(spaceCache cache.RespCache) bool {
		jsonBody, err := spaceCache.JsonBody()
		if err != nil {
			log.Printf(colors.ToRed("解析缓存响应体失败: %v"), err)
			return false
		}

		mediaSources, ok := jsonBody.Attr("MediaSources").Done()
		if !ok || mediaSources.Type() != jsons.JsonTypeArr || mediaSources.Empty() {
			return false
		}
		newMediaSources := jsons.NewEmptyArr()
		mediaSources.RangeArr(func(index int, value *jsons.Item) error {
			cacheId, err := url.QueryUnescape(value.Attr("Id").Val().(string))
			if err == nil && cacheId == reqId {
				newMediaSources.Append(value)
				updateCache(spaceCache, jsonBody, index)
				return jsons.ErrBreakRange
			}
			return nil
		})
		if newMediaSources.Empty() {
			return false
		}

		jsonBody.Put("MediaSources", newMediaSources)
		respHeader := spaceCache.Headers()
		respHeader.Del("Content-Length")
		https.CloneHeader(c, respHeader)
		c.JSON(http.StatusOK, jsonBody.Struct())
		return true
	}

	// 1 查询缓存空间
	spaceCache, ok := getPlaybackInfoByCacheSpace(itemInfo)
	if ok {
		// 未传递 MediaSourceId, 返回整个缓存数据
		if itemInfo.MsInfo.Empty {
			log.Printf(colors.ToBlue("复用缓存空间中的 PlaybackInfo 信息, itemId: %s"), itemInfo.Id)
			c.Status(spaceCache.Code())
			https.CloneHeader(c, spaceCache.Headers())
			// 避免缓存的请求头中出现脏数据
			c.Header("Access-Control-Allow-Origin", "*")
			c.Writer.Write(spaceCache.BodyBytes())
			c.Writer.Flush()
			return true
		}
		// 尝试从缓存中匹配指定的 MediaSourceId 信息
		if findMediaSourceAndReturn(spaceCache) {
			return true
		}
	}

	// 如果是全量查询, 从缓存中拿不到数据, 就触发手动请求全量
	if itemInfo.MsInfo.Empty {
		return false
	}

	// 一般在请求指定 MediaSourceId 的 PlaybackInfo 信息时必然会先请求一次全量的 PlaybackInfo 信息
	// 当用户停在已获取全量信息的剧集详情页面时缓存被清理, 或者是代理服务重启导致的缓存丢失
	// 此时再去播放就会触发这个错误提示, 需要等待下次全量请求完成之后才可以正常播放
	c.String(http.StatusInternalServerError, "查无缓存, 请稍后尝试重新播放")
	return true
}

// LoadCacheItems 拦截并代理 items 接口
//
// 如果 PlaybackInfo 缓存空间有相应的缓存
// 则将缓存中的 MediaSources 信息覆盖到响应中
//
// 防止转码资源信息丢失
func LoadCacheItems(c *gin.Context) {
	// 代理请求
	res, ok := proxyAndSetRespHeader(c)
	if !ok {
		return
	}
	resJson := res.Data
	defer func() {
		c.JSON(res.Code, resJson.Struct())
	}()

	// 未开启转码资源获取功能
	if !config.C.VideoPreview.Enable {
		return
	}

	// 只处理特定类型的 Items 响应
	itemType, _ := resJson.Attr("Type").String()
	if !ValidCacheItemsTypeRegex.MatchString(itemType) {
		return
	}

	// 特定客户端不处理
	if UnvalidCacheItemsUARegex.MatchString(c.GetHeader("User-Agent")) {
		return
	}

	// 解析出 ItemId
	itemInfo, err := resolveItemInfo(c)
	if err != nil {
		return
	}
	log.Printf(colors.ToBlue("itemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))

	// coverMediaSources 解析 PlaybackInfo 中的 MediaSources 属性
	// 并覆盖到当前请求的响应中
	// 成功覆盖时, 返回 true
	coverMediaSources := func(info *jsons.Item) bool {
		cacheMs, ok := info.Attr("MediaSources").Done()
		if !ok || cacheMs.Type() != jsons.JsonTypeArr {
			return false
		}
		log.Printf(colors.ToBlue("使用 PlaybackInfo 的 MediaSources 覆盖 Items 接口响应, itemId: %s"), itemInfo.Id)
		resJson.Put("MediaSources", cacheMs)
		c.Writer.Header().Del("Content-Length")
		return true
	}

	// 获取附带转码信息的 PlaybackInfo 数据
	spaceCache, ok := getPlaybackInfoByCacheSpace(itemInfo)
	if ok {
		cacheBody, err := spaceCache.JsonBody()
		if err != nil && coverMediaSources(cacheBody) {
			return
		}
	}

	// 缓存空间中没有当前 Item 的 PlaybackInfo 数据, 手动请求
	u := https.ClientRequestHost(c) + itemInfo.PlaybackInfoUri
	reqBody := io.NopCloser(bytes.NewBufferString(PlaybackCommonPayload))
	header := make(http.Header)
	header.Set("Content-Type", "text/plain")
	if itemInfo.ApiKeyType == Header {
		header.Set(HeaderFullAuthName, itemInfo.ApiKey)
	}
	resp, err := https.Request(http.MethodPost, u, header, reqBody)
	if err != nil {
		log.Printf(colors.ToRed("手动请求 PlaybackInfo 失败: %v, 不更新 Items 信息"), err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf(colors.ToRed("手动请求 PlaybackInfo 失败, code: %d, 不更新 Items 信息"), resp.StatusCode)
		return
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf(colors.ToRed("手动请求 PlaybackInfo 失败: %v, 不更新 Items 信息"), err)
		return
	}
	bodyJson, err := jsons.New(string(bodyBytes))
	if err != nil {
		log.Printf(colors.ToRed("手动请求 PlaybackInfo 失败: %v, 不更新 Items 信息"), err)
		return
	}
	coverMediaSources(bodyJson)
}

// calcPlaybackInfoSpaceCacheKey 根据请求的 item 信息计算 PlaybackInfo 在缓存空间中的 key
func calcPlaybackInfoSpaceCacheKey(itemInfo ItemInfo) string {
	return itemInfo.Id + "_" + itemInfo.ApiKey
}

// getPlaybackInfoByCacheSpace 从缓存空间中获取 PlaybackInfo 信息
func getPlaybackInfoByCacheSpace(itemInfo ItemInfo) (cache.RespCache, bool) {
	spaceCache, ok := cache.GetSpaceCache(PlaybackCacheSpace, calcPlaybackInfoSpaceCacheKey(itemInfo))
	if !ok {
		return nil, false
	}
	return spaceCache, true
}
