package emby

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
)

// TransferPlaybackInfo 代理 PlaybackInfo 接口, 防止客户端转码
func TransferPlaybackInfo(c *gin.Context) {
	// 1 解析资源信息
	itemInfo, err := resolveItemInfo(c)
	log.Printf(colors.ToBlue("ItemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))
	if checkErr(c, err) {
		return
	}

	defer func() {
		// 缓存 12h
		c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*12))
		// 将请求结果缓存到指定缓存空间下
		c.Header(cache.HeaderKeySpace, PlaybackCacheSpace)
		c.Header(cache.HeaderKeySpaceKey, itemInfo.Id+itemInfo.MsInfo.RawId)
	}()

	msInfo := itemInfo.MsInfo
	useTranscode := !msInfo.Empty && msInfo.Transcode
	if useTranscode {
		q := c.Request.URL.Query()
		q.Set("MediaSourceId", msInfo.OriginId)
		c.Request.URL.RawQuery = q.Encode()
	}

	// 2 请求 emby 源服务器的 PlaybackInfo 信息
	res, respHeader := RawFetch(c.Request.URL.String(), c.Request.Method, c.Request.Body)
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

		if ir, ok := source.Attr("IsRemote").Bool(); ok && ir {
			// 不阻塞远程资源
			respHeader.Del("Content-Length")
			https.CloneHeader(c, respHeader)
			c.JSON(res.Code, resJson.Struct())
			return haveReturned
		}

		// 转换直链链接
		source.Attr("SupportsDirectPlay").Set(true)
		source.Attr("SupportsDirectStream").Set(true)
		newUrl := fmt.Sprintf(
			"/videos/%s/stream?MediaSourceId=%s&%s=%s&Static=true",
			itemInfo.Id, source.Attr("Id").Val(), QueryApiKeyName, config.C.Emby.ApiKey,
		)

		// 简化资源名称
		name := findMediaSourceName(source)
		if name != "" {
			source.Attr("Name").Set(name)
		}
		name = source.Attr("Name").Val().(string)
		source.Attr("Name").Set(fmt.Sprintf("(原画) %s", name))

		if useTranscode {
			// 客户端请求指定的转码资源
			// 这里返回 emby 认得出的 master 响应
			tu, _ := url.Parse("/videos/" + itemInfo.Id + "/master.m3u8?DeviceId=a690fc29-1f3e-423b-ba23-f03049361a3b\u0026MediaSourceId=83ed6e4e3d820864a3d07d2ef9efab2e\u0026PlaySessionId=9f01e60a22c74ad0847319175912663b\u0026api_key=f53f3bf34c0543ed81415b86576058f2\u0026LiveStreamId=06044cf0e6f93cdae5f285c9ecfaaeb4_01413a525b3a9622ce6fdf19f7dde354_83ed6e4e3d820864a3d07d2ef9efab2e\u0026VideoCodec=h264,h265,hevc,av1\u0026AudioCodec=mp3,aac\u0026VideoBitrate=6808000\u0026AudioBitrate=192000\u0026AudioStreamIndex=1\u0026TranscodingMaxAudioChannels=2\u0026SegmentContainer=ts\u0026MinSegments=1\u0026BreakOnNonKeyFrames=True\u0026SubtitleStreamIndexes=-1\u0026ManifestSubtitles=vtt\u0026h264-profile=high,main,baseline,constrainedbaseline,high10\u0026h264-level=62\u0026hevc-codectag=hvc1,hev1,hevc,hdmv")
			q := tu.Query()
			q.Set("template_id", itemInfo.MsInfo.TemplateId)
			q.Set(QueryApiKeyName, config.C.Emby.ApiKey)
			q.Set("alist_path", itemInfo.MsInfo.AlistPath)
			tu.RawQuery = q.Encode()

			source.Attr("Name").Set(fmt.Sprintf("(%s) %s", msInfo.SourceNamePrefix, name))
			source.Put("SupportsTranscoding", jsons.NewByVal(true))
			source.Put("TranscodingUrl", jsons.NewByVal(tu.String()))
			source.Put("TranscodingSubProtocol", jsons.NewByVal("hls"))
			source.Put("TranscodingContainer", jsons.NewByVal("ts"))
			source.Put("SupportsDirectPlay", jsons.NewByVal(false))
			source.Put("SupportsDirectStream", jsons.NewByVal(false))
			source.DelKey("DirectStreamUrl")
			log.Printf(colors.ToBlue("设置转码播放链接为: %s"), tu.String())
			return nil
		}
		source.Attr("DirectStreamUrl").Set(newUrl)
		log.Printf(colors.ToBlue("设置直链播放链接为: %s"), newUrl)

		source.Put("SupportsTranscoding", jsons.NewByVal(false))
		source.DelKey("TranscodingUrl")
		source.DelKey("TranscodingSubProtocol")
		source.DelKey("TranscodingContainer")
		log.Println(colors.ToBlue("转码配置被移除"))

		// 添加转码 MediaSource 获取
		cfg := config.C.VideoPreview
		if !msInfo.Empty || !cfg.Enable || !cfg.ContainerValid(source.Attr("Container").Val().(string)) {
			return nil
		}
		resChan := make(chan []*jsons.Item, 1)
		go findVideoPreviewInfos(source, name, resChan)
		resChans = append(resChans, resChan)
		return nil
	})

	if err == haveReturned {
		return
	}

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

// LoadCacheItems 拦截并代理 items 接口
//
// 如果 PlaybackInfo 缓存空间有相应的缓存
// 则将缓存中的 MediaSources 信息覆盖到响应中
//
// 防止转码资源信息丢失
func LoadCacheItems(c *gin.Context) {
	// 1 代理请求
	res, ok := proxyAndSetRespHeader(c)
	if !ok {
		return
	}
	resJson := res.Data
	defer func() {
		c.JSON(res.Code, resJson.Struct())
	}()

	// 2 未开启转码资源获取功能
	if !config.C.VideoPreview.Enable {
		return
	}

	// 3 解析出 ItemId
	itemInfo, err := resolveItemInfo(c)
	if err != nil {
		return
	}
	log.Printf(colors.ToBlue("itemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))

	// 4 获取附带转码信息的 PlaybackInfo 数据
	cacheBody, ok := getPlaybackInfoByCacheSpace(itemInfo)
	if !ok {
		return
	}
	log.Println(colors.ToBlue("使用缓存空间中的 MediaSources 覆盖原始响应"))

	cacheMs, ok := cacheBody.Attr("MediaSources").Done()
	if !ok || cacheMs.Type() != jsons.JsonTypeArr {
		return
	}

	// 5 覆盖原始响应
	resJson.Put("MediaSources", cacheMs)
	c.Writer.Header().Del("Content-Length")
}

// getPlaybackInfoByCacheSpace 从缓存空间中获取 PlaybackInfo 信息
func getPlaybackInfoByCacheSpace(itemInfo ItemInfo) (*jsons.Item, bool) {
	spaceCache, ok := cache.GetSpaceCache(PlaybackCacheSpace, itemInfo.Id+itemInfo.MsInfo.RawId)
	if !ok {
		return nil, false
	}

	// 5 获取缓存响应体内容
	cacheBody, err := spaceCache.JsonBody()
	if err != nil {
		return nil, false
	}
	return cacheBody, true
}
